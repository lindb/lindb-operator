/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	alpha1 "github.com/lindb/lindb-operator/api/v1alpha1"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=lindb.lindb.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=lindb.lindb.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=lindb.lindb.io,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var lindbCluster = alpha1.Cluster{}
	err := r.Get(ctx, req.NamespacedName, &lindbCluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	if apierrors.IsNotFound(err) ||
		lindbCluster.GetDeletionTimestamp() != nil {
		logger.Info("Lindb cluster reconcile delete", "cluster", lindbCluster.Namespace+"/"+lindbCluster.Name)
		r.reconcileDelete(ctx, &lindbCluster)
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling Lindb cluster", "cluster", lindbCluster.Namespace+"/"+lindbCluster.Name)

	var cm = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lindbCluster.Name + "-config",
			Namespace: lindbCluster.Namespace,
		},
		Data: map[string]string{
			"empty.toml": "",
		},
	}

	if err := r.Create(ctx, &cm); !apierrors.IsAlreadyExists(err) {
		logger.Error(err, "Creating configmap failed", "configmap", lindbCluster.Namespace+"/"+lindbCluster.Name)
	}

	if err := r.reconcileBroker(ctx, &lindbCluster); err != nil {
		logger.Error(err, "Reconciling broker failed", "broker", lindbCluster.Namespace+"/"+lindbCluster.Name)
	}

	if err := r.reconcileStorage(ctx, &lindbCluster); err != nil {
		logger.Error(err, "Reconciling storage failed", "storage", lindbCluster.Namespace+"/"+lindbCluster.Name)
	}

	logger.Info("Reconciling Lindb cluster done", "cluster", lindbCluster.Namespace+"/"+lindbCluster.Name)

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) reconcileBroker(ctx context.Context, cluster *alpha1.Cluster) error {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Reconciling broker", "broker", cluster.Namespace+"/"+cluster.Name)

	var image string
	if cluster.Spec.Storages.Image != "" {
		image = cluster.Spec.Storages.Image
	} else {
		image = cluster.Spec.Image
	}

	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-broker",
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: cluster.Spec.Brokers.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app-name": cluster.Name + "-broker",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cluster.Name + "-broker",
					Namespace: cluster.Namespace,
					Labels: map[string]string{
						"app-name": cluster.Name + "-broker",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "broker",
							Image: image,
							Command: []string{
								"/usr/bin/lind",
							},
							Args: []string{
								"broker",
								"run",
								"--config=/etc/lindb/empty.toml",
							},
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: cluster.Spec.Brokers.HttpPort,
								},
								{
									ContainerPort: cluster.Spec.Brokers.GrpcPort,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LINDB_COORDINATOR_NAMESPACE",
									Value: cluster.Spec.EtcdNamespace,
								},
								{
									Name:  "LINDB_COORDINATOR_ENDPOINTS",
									Value: fmt.Sprintf("%s", strings.Join(cluster.Spec.EtcdEndpoints, ",")),
								},
								{
									Name:  "LINDB_BROKER_HTTP_PORT",
									Value: fmt.Sprintf("%d", cluster.Spec.Brokers.HttpPort),
								},
								{
									Name:  "LINDB_BROKER_GRPC_PORT",
									Value: fmt.Sprintf("%d", cluster.Spec.Brokers.GrpcPort),
								},
								{
									Name:  "LINDB_MONITOR_REPORT_INTERVAL",
									Value: cluster.Spec.Brokers.Monitor.ReportInterval,
								},
								{
									Name:  "LINDB_MONITOR_URL",
									Value: cluster.Spec.Brokers.Monitor.ReportUrl,
								},
								{
									Name:  "LINDB_LOGGING_DIR",
									Value: cluster.Spec.Brokers.Logging.Dir + "/broker",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/lindb",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + "-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := r.Create(ctx, &deploy)
	if err != nil {
		logger.Error(err, "Create deployment failed", "deployment", deploy.Namespace+"/"+deploy.Name)
	}

	return nil
}

func (r *ClusterReconciler) reconcileStorage(ctx context.Context, cluster *alpha1.Cluster) error {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Reconciling storage", "storage", cluster.Namespace+"/"+cluster.Name)

	var image string
	if cluster.Spec.Storages.Image != "" {
		image = cluster.Spec.Storages.Image
	} else {
		image = cluster.Spec.Image
	}

	var sts = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-storage",
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            cluster.Spec.Storages.Replicas,
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app-name": cluster.Name + "-storage",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cluster.Name + "-storage",
					Namespace: cluster.Namespace,
					Labels: map[string]string{
						"app-name": cluster.Name + "-storage",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "storage",
							Image: image,
							Command: []string{
								"/usr/bin/lind",
							},
							Args: []string{
								"storage",
								"run",
								"--config=/etc/lindb/empty.toml",
							},
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: cluster.Spec.Storages.HttpPort,
								},
								{
									ContainerPort: cluster.Spec.Storages.GrpcPort,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "LINDB_COORDINATOR_NAMESPACE",
									Value: cluster.Spec.EtcdNamespace,
								},
								{
									Name:  "LINDB_COORDINATOR_ENDPOINTS",
									Value: fmt.Sprintf("[%s]", strings.Join(cluster.Spec.EtcdEndpoints, ",")),
								},
								{
									Name:  "LINDB_STORAGE_HTTP_PORT",
									Value: fmt.Sprintf("%d", cluster.Spec.Storages.HttpPort),
								},
								{
									Name:  "LINDB_STORAGE_GRPC_PORT",
									Value: fmt.Sprintf("%d", cluster.Spec.Storages.GrpcPort),
								},
								{
									Name:  "LINDB_STORAGE_WAL_DIR",
									Value: cluster.Spec.Storages.Logging.Dir + "$POD_NAME" + "/wal",
								},
								{
									Name:  "LINDB_STORAGE_TSDB_DIR",
									Value: cluster.Spec.Storages.Logging.Dir + "$POD_NAME" + "/tsdb",
								},
								{
									Name:  "LINDB_MONITOR_REPORT_INTERVAL",
									Value: "10s",
								},
								{
									Name:  "LINDB_STORAGE_WAL_REMOVE_TASK_INTERVAL",
									Value: "1m",
								},
								{
									Name:  "LINDB_MONITOR_URL",
									Value: cluster.Spec.Storages.Monitor.ReportUrl,
								},
								{
									Name:  "LINDB_LOGGING_DIR",
									Value: cluster.Spec.Storages.Logging.Dir + "/storage",
								},
								{
									Name:  "LINDB_LOGGING_LEVEL",
									Value: "debug",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/lindb",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + "-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := r.Create(ctx, &sts)
	if err != nil {
		logger.Error(err, "Create statefulset failed", "statefulset", sts.Namespace+"/"+sts.Name)
	}

	return nil
}

func (r *ClusterReconciler) reconcileDelete(ctx context.Context, cluster *alpha1.Cluster) error {
	if r.reconcileDeleteBroker(ctx, cluster) != nil {
		return nil
	}

	if r.reconcileDeleteStorage(ctx, cluster) != nil {
		return nil
	}
	return nil
}

func (r *ClusterReconciler) reconcileDeleteBroker(ctx context.Context, cluster *alpha1.Cluster) error {
	return nil
}

func (r *ClusterReconciler) reconcileDeleteStorage(ctx context.Context, cluster *alpha1.Cluster) error {
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&alpha1.Cluster{}).
		Complete(r)
}
