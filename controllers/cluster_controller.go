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
	"net/url"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete

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
	logger.Info("Reconciling Lindb cluster", "cluster", lindbCluster.Namespace+"/"+lindbCluster.Name)

	if apierrors.IsNotFound(err) || lindbCluster.GetDeletionTimestamp() != nil {
		return r.reconcileDelete(ctx, &lindbCluster)
	}

	return r.reconcileNormal(ctx, &lindbCluster)
}

func (r *ClusterReconciler) reconcileNormal(ctx context.Context, cluster *alpha1.Cluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Reconciling normal", "cluster", cluster.Namespace+"/"+cluster.Name)

	if err := r.reconcileDepend(ctx, cluster); err != nil {
		logger.Error(err, "Reconciling depend failed", "depend", cluster.Namespace+"/"+cluster.Name)
		return ctrl.Result{}, err
	}

	if err := r.reconcileBroker(ctx, cluster); err != nil {
		logger.Error(err, "Reconciling broker failed", "broker", cluster.Namespace+"/"+cluster.Name)
		return ctrl.Result{}, err
	}

	if err := r.reconcileStorage(ctx, cluster); err != nil {
		logger.Error(err, "Reconciling storage failed", "storage", cluster.Namespace+"/"+cluster.Name)
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling Lindb cluster successfully", "cluster", cluster.Namespace+"/"+cluster.Name)

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) reconcileDepend(ctx context.Context, cluster *alpha1.Cluster) error {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Reconciling depend", "depend", cluster.Namespace+"/"+cluster.Name)

	var cm = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-configmap",
			Namespace: cluster.Namespace,
			Labels:    setResourceClusterLabels(nil, cluster),
		},
		Data: map[string]string{
			"empty.toml": "",
		},
	}

	if err := r.Create(ctx, &cm); err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error(err, "Creating configmap failed", "configmap", cluster.Namespace+"/"+cluster.Name)
		return err
	}

	logger.Info("Reconciling configmap successfully", "depend", cluster.Namespace+"/"+cluster.Name)

	var gp2Sc = "gp2"

	var pvc = corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-pvc",
			Namespace: cluster.Namespace,
			Labels:    setResourceClusterLabels(nil, cluster),
		},
		// support tag based storage on ec2
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &gp2Sc,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse(cluster.Spec.Cloud.Storage.StorageSize),
				},
			},
			VolumeMode: &[]corev1.PersistentVolumeMode{corev1.PersistentVolumeFilesystem}[0],
		},
	}

	if err := r.Create(ctx, &pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error(err, "Creating pvc failed", "pvc", cluster.Namespace+"/"+cluster.Name)
		return err
	}

	return nil
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

	var label = map[string]string{
		"app-name": "lindb-broker",
	}

	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-broker",
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: cluster.Spec.Brokers.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: setResourceClusterLabels(label, cluster),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cluster.Name + "-broker",
					Namespace: cluster.Namespace,
					Labels:    setResourceClusterLabels(label, cluster),
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
							TTY:             true,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: cluster.Spec.Brokers.HttpPort,
								},
								{
									ContainerPort: cluster.Spec.Brokers.GrpcPort,
								},
							},
						},
					},
					DNSPolicy: corev1.DNSClusterFirst,
				},
			},
		},
	}

	injectEnv2PodTemplate(ctx, &deploy.Spec.Template, cluster)
	injectVolume2PodTemplate(ctx, &deploy.Spec.Template, cluster)

	if err := r.Create(ctx, &deploy); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return r.Update(ctx, &deploy)
		}
		logger.Error(err, "Create deployment failed", "deployment", deploy.Namespace+"/"+deploy.Name)
	}

	logger.Info("Reconciling deployment successfully", "deployment", deploy.Namespace+"/"+deploy.Name)

	// use headless service to expose broker
	var svc = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-svc",
			Namespace: cluster.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  setResourceClusterLabels(label, cluster),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       cluster.Spec.Brokers.HttpPort,
					TargetPort: intstr.FromInt(int(cluster.Spec.Brokers.HttpPort)),
				},
			},
		},
	}

	if err := r.Create(ctx, &svc); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return r.Update(ctx, &svc)
		}
		logger.Error(err, "Create service failed", "service", svc.Namespace+"/"+svc.Name)
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

	var label = map[string]string{
		"app-name": "lindb-storage",
	}

	var sts = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-storage",
			Namespace: cluster.Namespace,
			Labels:    setResourceClusterLabels(label, cluster),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            cluster.Spec.Storages.Replicas,
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: setResourceClusterLabels(label, cluster),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cluster.Name + "-storage",
					Namespace: cluster.Namespace,
					Labels:    setResourceClusterLabels(label, cluster),
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
							TTY:             true,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: cluster.Spec.Storages.HttpPort,
								},
								{
									ContainerPort: cluster.Spec.Storages.GrpcPort,
								},
							},
						},
					},
					DNSPolicy: corev1.DNSClusterFirst,
				},
			},
		},
	}

	injectEnv2PodTemplate(ctx, &sts.Spec.Template, cluster)
	injectVolume2PodTemplate(ctx, &sts.Spec.Template, cluster)
	injectCreateStorage(ctx, &sts.Spec.Template, cluster)
	injectCreateInternalDatabase(ctx, &sts.Spec.Template, cluster)

	err := r.Create(ctx, &sts)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return r.Update(ctx, &sts)
		}
		logger.Error(err, "Create statefulset failed", "statefulset", sts.Namespace+"/"+sts.Name)
	}

	logger.Info("Reconciling statefulset successfully", "statefulset", sts.Namespace+"/"+sts.Name)

	return nil
}

func (r *ClusterReconciler) reconcileDelete(ctx context.Context, cluster *alpha1.Cluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "phase", "reconcileDelete")
	logger.Info("Reconcile delete cluster", "cluster", cluster.Namespace+"/"+cluster.Name)

	if err := r.reconcileDependResource(ctx, cluster); err != nil {
		logger.Error(err, "Delete depend resource failed", "cluster", cluster.Namespace+"/"+cluster.Name)
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeleteBroker(ctx, cluster); err != nil {
		logger.Error(err, "Delete broker failed", "cluster", cluster.Namespace+"/"+cluster.Name)
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeleteStorage(ctx, cluster); err != nil {
		logger.Error(err, "Delete storage failed", "cluster", cluster.Namespace+"/"+cluster.Name)
		return ctrl.Result{}, err
	}

	logger.Info("Reconcile delete cluster success", "cluster", cluster.Namespace+"/"+cluster.Name)

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) reconcileDependResource(ctx context.Context, cluster *alpha1.Cluster) error {
	logger := log.FromContext(ctx, "phase", "reconcileDependResource")

	logger.Info("Reconcile delete depend resource", "cluster", cluster.Namespace+"/"+cluster.Name)

	// 1. delete configmap
	logger.V(4).Info("Delete configmap", "configmap", cluster.Namespace+"/"+cluster.Name+"-config")
	if err := r.Delete(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-configmap",
			Namespace: cluster.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Delete configmap failed", "configmap", cluster.Namespace+"/"+cluster.Name+"-config")
		return err
	}

	// 2. delete log pvc
	logger.V(4).Info("Delete pvc", "pvc", cluster.Namespace+"/"+cluster.Name+"-pvc")
	if err := r.Delete(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-pvc",
			Namespace: cluster.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Delete pvc failed", "pvc", cluster.Namespace+"/"+cluster.Name+"-pvc")
		return err
	}

	logger.Info("Reconcile delete depend resource success", "cluster", cluster.Namespace+"/"+cluster.Name)
	return nil
}

func (r *ClusterReconciler) reconcileDeleteBroker(ctx context.Context, cluster *alpha1.Cluster) error {
	logger := log.FromContext(ctx, "phase", "reconcileDeleteBroker")
	logger.Info("Reconcile delete broker", "cluster", cluster.Namespace+"/"+cluster.Name)

	logger.V(4).Info("Delete Deployment", "Deployment", cluster.Namespace+"/"+cluster.Name+"-broker")
	if err := r.Delete(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-broker",
			Namespace: cluster.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Delete Deployment failed", "Deployment", cluster.Namespace+"/"+cluster.Name+"-broker")
		return err
	}

	logger.Info("Reconcile delete broker success", "cluster", cluster.Namespace+"/"+cluster.Name)
	return nil
}

func (r *ClusterReconciler) reconcileDeleteStorage(ctx context.Context, cluster *alpha1.Cluster) error {
	logger := log.FromContext(ctx, "phase", "reconcileDeleteStorage")
	logger.Info("Reconcile delete storage", "cluster", cluster.Namespace+"/"+cluster.Name)

	if err := r.Delete(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-storage",
			Namespace: cluster.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Delete statefulset failed", "statefulset", cluster.Namespace+"/"+cluster.Name+"-storage")
		return err
	}

	logger.Info("Reconcile delete storage success", "cluster", cluster.Namespace+"/"+cluster.Name)
	return nil
}

func injectEnv2PodTemplate(ctx context.Context, temp *corev1.PodTemplateSpec, cluster *alpha1.Cluster) {
	temp.Spec.Containers[0].Env = []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},

		// general
		{
			Name:  "LINDB_COORDINATOR_NAMESPACE",
			Value: cluster.Spec.EtcdNamespace,
		},
		{
			Name:  "LINDB_COORDINATOR_ENDPOINTS",
			Value: fmt.Sprintf("%s", strings.Join(cluster.Spec.EtcdEndpoints, ",")),
		},

		// storage
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
			Value: cluster.Spec.Cloud.Storage.MountPath + "/$(POD_NAME)/wal",
		},
		{
			Name:  "LINDB_STORAGE_TSDB_DIR",
			Value: cluster.Spec.Cloud.Storage.MountPath + "/$(POD_NAME)/tsdb",
		},
		{
			Name:  "LINDB_STORAGE_WAL_REMOVE_TASK_INTERVAL",
			Value: "1m",
		},

		// broker
		{
			Name:  "LINDB_BROKER_HTTP_PORT",
			Value: fmt.Sprintf("%d", cluster.Spec.Brokers.HttpPort),
		},
		{
			Name:  "LINDB_BROKER_GRPC_PORT",
			Value: fmt.Sprintf("%d", cluster.Spec.Brokers.GrpcPort),
		},

		// monitor
		{
			Name:  "LINDB_MONITOR_REPORT_INTERVAL", // TODO: add to spec
			Value: cluster.Spec.Brokers.Monitor.ReportInterval,
		},
		{
			Name:  "LINDB_MONITOR_URL",
			Value: cluster.Spec.Storages.Monitor.ReportUrl,
		},
		{
			Name:  "LINDB_LOGGING_DIR",
			Value: cluster.Spec.Cloud.Storage.MountPath + "/$(POD_NAME)/logs",
		},
		{
			Name:  "LINDB_LOGGING_LEVEL",
			Value: "debug",
		},
	}
}

func injectCreateStorage(ctx context.Context, temp *corev1.PodTemplateSpec, cluster *alpha1.Cluster) {
	log := log.FromContext(ctx, "phase", "injectCreateStorage")

	log.V(4).Info("Inject StartupProbe", "cluster", cluster.Namespace+"/"+cluster.Name)

	params := url.Values{}
	params.Set("sql", `create storage {\"config\":{\"namespace\":\"/lindb-storage\",\"timeout\":10,\"dialTimeout\":10,\"leaseTTL\":10,\"endpoints\":[\"http://etcd:2379\"]}}`)
	var path = "api/v1/exec?"
	path += params.Encode()

	temp.Spec.Containers[0].StartupProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: cluster.Name + "-svc",
				Path: path,
				Port: intstr.FromInt(int(cluster.Spec.Brokers.HttpPort)),
			},
		},
		InitialDelaySeconds: 5,
		TimeoutSeconds:      10,
		PeriodSeconds:       10,
	}
}

func injectCreateInternalDatabase(ctx context.Context, temp *corev1.PodTemplateSpec, cluster *alpha1.Cluster) {
	log := log.FromContext(ctx, "phase", "injectCreateInternalDatabase")

	log.V(4).Info("Inject post start", "cluster", cluster.Namespace+"/"+cluster.Name)

	params := url.Values{}
	params.Set("sql", `create database {\"option\":{\"intervals\":[{\"interval\":\"10s\",\"retention\":\"30d\"},{\"interval\":\"5m\",\"retention\":\"3M\"},{\"interval\":\"1h\",\"retention\":\"2y\"}],\"autoCreateNS\":true,\"behead\":\"1h\",\"ahead\":\"1h\"},\"name\":\"_internal\",\"storage\":\"/lindb-storage\",\"numOfShard\":3,\"replicaFactor\":2}`)
	var path = "api/v1/exec?"
	path += params.Encode()

	temp.Spec.Containers[0].StartupProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: cluster.Name + "-svc",
				Path: path,
				Port: intstr.FromInt(int(cluster.Spec.Brokers.HttpPort)),
			},
		},
		InitialDelaySeconds: 5,
		TimeoutSeconds:      10,
		PeriodSeconds:       10,
	}
}

func injectVolume2PodTemplate(ctx context.Context, temp *corev1.PodTemplateSpec, cluster *alpha1.Cluster) {
	temp.Spec.Volumes = []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cluster.Name + "-configmap",
					},
				},
			},
		},
		{
			Name: "storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: cluster.Name + "-pvc",
				},
			},
		},
	}

	temp.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/etc/lindb",
		},
		{
			Name:      "storage",
			MountPath: cluster.Spec.Cloud.Storage.MountPath,
		},
	}
}

func setResourceClusterLabels(labels map[string]string, cluster *alpha1.Cluster) map[string]string {
	var ret = make(map[string]string)

	if labels != nil {
		for k, v := range labels {
			ret[k] = v
		}
	}

	for k, v := range cluster.Labels {
		ret[k] = v
	}

	ret["lindb.lindb.io/cluster"] = cluster.Name

	return ret
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&alpha1.Cluster{}).
		Complete(r)
}
