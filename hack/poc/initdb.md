## create 


1. crate storage

```shell
curl -G http://localhost:9000/api/v1/exec --data-urlencode "sql=create storage {\"config\":{\"namespace\":\"/lindb-storage\",\"timeout\":10,\"dialTimeout\":10,\"leaseTTL\":10,\"endpoints\":[\"http://etcd:2379\"]}}"
```


2. create database

```shell
curl -G http://localhost:9000/api/v1/exec --data-urlencode "sql=create database {\"option\":{\"intervals\":[{\"interval\":\"10s\",\"retention\":\"30d\"},{\"interval\":\"5m\",\"retention\":\"3M\"},{\"interval\":\"1h\",\"retention\":\"2y\"}],\"autoCreateNS\":true,\"behead\":\"1h\",\"ahead\":\"1h\"},\"name\":\"_internal\",\"storage\":\"/lindb-storage\",\"numOfShard\":3,\"replicaFactor\":2}"
```

3. show database

```shell
curl -G http://localhost:9000/api/v1/exec --data-urlencode "sql=show databases"
```