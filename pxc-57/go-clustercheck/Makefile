MYSQL_ROOT_PASSWORD?=secret
MYSQL_CHECK_HOST?=127.0.0.1:3306

AVAILABLE_WHEN_DONNOR: defaults to no (empty)
	AVAILABLE_WHEN_READONLY: defaults to -1 (not available)
	docker run --rm -v /Users/sdelrio/docker/cluster-sql/percona-doco
build:
	docker run --rm -v $(PWD):/go -w /go golang env
	docker run --rm -v $(PWD):/go -w /go golang go install clustercheck
clean:
	rm -f bin/clustercheck
get:
	docker run --rm -v $(PWD):/go -w /go golang go get clustercheck
test:
	docker run --rm -v $(PWD):/go -w /go golang ls -l bin/clustercheck
	docker run --rm -v $(PWD):/go -w /go golang bin/clustercheck -h
	docker run --rm -v $(PWD):/go -w /go golang bin/clustercheck

