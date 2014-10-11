QDo
======================
A queue implementation written in Golang. Currently using leveldb library as
storage backend.


# Disclaimer: this is a toy project.

# Usage
Create queue

    curl http://127.0.0.1:7999/api/queue \
       -d queue_id=foo \
       -d max_concurrent=2 \
       -d max_rate=100 \
       -d task_timeout=60 \
       -d task_max_tries=3

Delete queue

    curl -X DELETE http://127.0.0.1:7999/api/queue/foo

Create task

    curl http://127.0.0.1:7999/api/queue/foo/task \
       -d target=http://127.0.0.1/mytask \
       -d "payload={'foo': 'bar'}"

Create scheduled task

    curl http://127.0.0.1:8080/api/queue/foo/task \
       -d target=http://127.0.0.1/mytask \
       -d scheduled=1399999999 \
       -d "payload={'foo': 'bar'}"

Delete all tasks

    curl -X DELETE http://127.0.0.1:8080/api/queue/foo/task

Get queue stats

    curl http://127.0.0.1:8080/api/queue/foo/stats


# Build binfile with go-bindata
## Install
go get github.com/jteeuwen/go-bindata/...
cd /gopath/src/github.com/jteeuwen/go-bindata
go install

## Generate binfile
cd http
go-bindata -pkg http static/ static/fonts template

