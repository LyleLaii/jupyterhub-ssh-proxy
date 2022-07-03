# A simple build script, just for specify version info
CGO_ENABLED=1 GOOS=linux  go build -o out/jupyterhub-ssh-proxy \
        -ldflags "-X github.com/lylelaii/golang_utils/version/v1.Version=`cat VERSION` -X github.com/lylelaii/golang_utils/version/v1.Revision=`git rev-parse HEAD` -X github.com/lylelaii/golang_utils/version/v1.Branch=`git rev-parse --abbrev-ref HEAD` -X github.com/lylelaii/golang_utils/version/v1.BuildUser=`whoami` -X github.com/lylelaii/golang_utils/version/v1.BuildDate=`date +%Y%m%d-%H:%M:%S`"  \
        -v -a -trimpath cmd/proxy/main.go