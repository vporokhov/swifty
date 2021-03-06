#!/bin/bash
if [ "x${GOPATH}" != "x$(pwd)/vendor:$(pwd)" ]; then
	echo "Set GOPATH to $(pwd)/vendor:$(pwd)"
	exit 1
fi

VGOPATH="$(pwd)/vendor"

if [ -d "${VGOPATH}/src" ]; then
	echo "Vendor is populated"
	exit 0
fi

set -x
set -e

# We need 6.0.0 version of the k8s client libs. When built, the lib
# gets the protobuf library of some given version, which is SUDDENLY
# incompatible with prometheus client lib. The latter need protobuf
# of version 1.1.0. Thus, we first download the k8s, then checkout
# it to 6.0.0, then fetch the Godep-s of it, then fiv protobuf version
# to be 1.1.1, then install k8s, then proceed with the rest.

if which yum ; then
	yum install -y golang patch librados2-devel glibc-headers glibc-static
	yum groupinstall -y "Development Libraries" 
elif which apt-get ; then
	apt-get install -y golang librados-dev
fi

go get github.com/tools/godep
if [ "x$USER" = "xroot" ] ; then
	cp ${VGOPATH}/bin/godep /usr/bin
else
	case :$PATH: # notice colons around the value
		in *:$HOME/bin:*) ;; # do nothing, it's there
		*) echo "$HOME/bin not in $PATH" >&2; exit 0 ;;
	esac
	cp ${VGOPATH}/bin/godep $HOME/bin
fi

go get -d k8s.io/client-go/...
cd ${VGOPATH}/src/k8s.io/client-go
git checkout -b swy6.0.0 v6.0.0
godep restore ./...
cd -
git -C ${VGOPATH}/src/github.com/golang/protobuf checkout -b swy1.1.0 v1.1.0
go install k8s.io/client-go/...
go get github.com/prometheus/client_golang/prometheus
go get github.com/go-sql-driver/mysql
go get github.com/gorilla/mux
go get github.com/gorilla/websocket
go get gopkg.in/yaml.v2
go get github.com/michaelklishin/rabbit-hole
go get github.com/streadway/amqp
go get go.uber.org/zap
go get gopkg.in/mgo.v2
go get -d gopkg.in/robfig/cron.v2;
patch -d${VGOPATH}/src/gopkg.in/robfig/cron.v2 -p1 < $(pwd)/contrib/robfig-cron.patch;
go install gopkg.in/robfig/cron.v2
go get code.cloudfoundry.org/bytefmt
go get github.com/ceph/go-ceph/rados # this gent is broken in deb, so last
