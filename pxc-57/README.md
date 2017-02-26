Percona XtraDB Cluster docker image
===================================

The docker image is available right now at `percona/percona-xtradb-cluster:5.7`.
The image supports work in Docker Network, including overlay networks,
so that you can install Percona XtraDB Cluster nodes on different boxes.
There is an initial support for the etcd discovery service.

Basic usage
-----------

For an example, see the `start_node.sh` script.

The `CLUSTER_NAME` environment variable should be set, and the easiest to do it is:
`export CLUSTER_NAME=cluster1`

The script will try to create an overlay network `${CLUSTER_NAME}_net`.
If you want to have a bridge network or network with a specific parameter,
create it in advance.
For example:
`docker network create -d bridge ${CLUSTER_NAME}_net`

The Docker image accepts the following parameters:
* One of `MYSQL_ROOT_PASSWORD`, `MYSQL_ALLOW_EMPTY_PASSWORD` or `MYSQL_RANDOM_ROOT_PASSWORD` must be defined
* The image will create the user `xtrabackup@localhost` for the XtraBackup SST method. If you want to use a password for the `xtrabackup` user, set `XTRABACKUP_PASSWORD`. 
* If you want to use swarm service, set the service name to `SWARM_SERVICE`. The image will automatically find a running cluser by `CLUSTER_NAME` and join to the existing cluster (or start a new one).
* If you want to start without the discovery service, use the `CLUSTER_JOIN` variable. Empty variables will start a new cluster, To join an existing cluster, set `CLUSTER_JOIN` to the list of IP addresses running cluster nodes.


Discovery service
-----------------

The cluster will try to register itself in the discovery service, so that new nodes or ProxySQL can easily find running nodes.

The image will look for IPs of sql-cluster and remove his own IP for the join parameter of galera cluster.

Example service with swarm:

```
NAME=sdelrio/percona-docker
VERSION=latest
SERVICE_NAME=sql-cluster
CLUSTER_NAME=cluster1
ROOT_PASS=secret
XDB_PASS=secret
NETWORK=cluster1_net

docker network create $NETWORK -d overlay

docker service create -p 3306 \
    --update-parallelism 1 \
    --update-delay 60s \
    --network $NETWORK \
    --name $SERVICE_NAME \
    -e MYSQL_ROOT_PASSWORD=$ROOT_PASS \
    -e CLUSTER_NAME=$CLUSTER_NAME \
    -e SWARM_SERVICE=$SERVICE_NAME \
    -e XTRABACKUP_PASSWORD=$XDB_PASS \
    $NAME:$VERSION
```

------------------------------

The following link is a great introduction with easy steps on how to run a Docker overlay network: http://chunqi.li/2015/11/09/docker-multi-host-networking/


Running with ProxySQL
---------------------

The ProxySQL image https://hub.docker.com/r/perconalab/proxysql/
provides an integration with Percona XtraDB Cluster and discovery service.

You can start proxysql image by
```
docker run -d -p 3306:3306 -p 6032:6032 --net=$NETWORK_NAME --name=${CLUSTER_NAME}_proxysql \
        -e CLUSTER_NAME=$CLUSTER_NAME \
        -e ETCD_HOST=$ETCD_HOST \
        -e MYSQL_ROOT_PASSWORD=Theistareyk \
        -e MYSQL_PROXY_USER=proxyuser \
        -e MYSQL_PROXY_PASSWORD=s3cret \
        perconalab/proxysql
```

where `MYSQL_ROOT_PASSWORD` is the root password for the MySQL nodes. The password is needed to register the proxy user. The user `MYSQL_PROXY_USER` with password `MYSQL_PROXY_PASSWORD` will be registered on all Percona XtraDB Cluster nodes.


Running `docker exec -it ${CLUSTER_NAME}_proxysql add_cluster_nodes.sh` will register all nodes in the ProxySQL.

