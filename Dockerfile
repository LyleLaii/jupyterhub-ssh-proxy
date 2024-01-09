FROM golang:1.16.4-buster AS go-builder

WORKDIR /jupyterhub-ssh-proxy
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=1 GOOS=linux go build -a -trimpath \
    -o bin/jupyterhub-ssh-proxy \
    -ldflags "-X github.com/lylelaii/golang_utils/version/v1.Version=`cat VERSION` -X github.com/lylelaii/golang_utils/version/v1.Revision=`git rev-parse HEAD` -X github.com/lylelaii/golang_utils/version/v1.Branch=`git rev-parse --abbrev-ref HEAD` -X github.com/lylelaii/golang_utils/version/v1.BuildUser=`whoami` -X github.com/lylelaii/golang_utils/version/v1.BuildDate=`date +%Y%m%d-%H:%M:%S`"  \
    cmd/proxy/main.go

RUN ssh-keygen -q -N "" -f ./etc/id_rsa

FROM alpine:3.12

LABEL MAINTAINER=lyle.lai

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /opt/glibc
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.34-r0/glibc-2.34-r0.apk && \
    apk add glibc-2.34-r0.apk

WORKDIR /opt/jupyterhub-ssh-proxy

COPY --from=go-builder /jupyterhub-ssh-proxy/bin /opt/jupyterhub-ssh-proxy/bin
RUN mkdir /opt/jupyterhub-ssh-proxy/logs && \
    mkdir /opt/jupyterhub-ssh-proxy/etc && \
    rm -rf /opt/glibc
COPY --from=go-builder /jupyterhub-ssh-proxy/etc/id_rsa* /opt/jupyterhub-ssh-proxy/etc/
COPY etc/config.yaml etc/config.yaml

EXPOSE 8080

ENTRYPOINT ["./bin/jupyterhub-ssh-proxy"]