# lindb-operator

LinDB Operator creates/configures/manages LinDB clusters atop Kubernetes.


## create

to create the lindb by the following yaml file.

```yaml
apiVersion: lindb.lindb.io/v1alpha1
kind: Cluster
metadata:
  labels:
    lindb.lindb.io/tenant: lindb-user1
  name: cluster-sample
spec:
  paused: false
  etcdNamespace: /lindb-cluster
  etcdEndpoints:
    - http://etcd:2379
  image: wangguohao/lindb:v0.2.1
  cloud:
    type: aws
    storage:
      type: shared
      storageClass: gp2
      storageSize: 10Gi
      mountPath: /data
  brokers:
    replicas: 1
    httpPort: 9000
    grpcPort: 9001
    monitor:
      reportInterval: 10s
      url: http://localhost:9000/api/v1/write?db=_internal
  storages:
    replicas: 1
    grpcPort: 2891
    httpPort: 2892
    monitor:
      reportInterval: 10s
      url: http://localhost:9000/api/v1/write?db=_internal
```

you can find one broker and one storage. and to query the `_internal` databases by login in the borker pod.

```shell
curl -v -G http://localhost:9000/api/v1/exec --data-urlencode "sql=show databases"
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 9000 (#0)
> GET /api/v1/exec?sql=show%20databases HTTP/1.1
> Host: localhost:9000
> User-Agent: curl/7.61.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Fri, 24 Mar 2023 08:48:34 GMT
< Content-Length: 13
<
* Connection #0 to host localhost left intact
["_internal"]
```