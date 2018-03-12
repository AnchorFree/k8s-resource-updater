FROM golang:1.8

COPY . /go/src/github.com/anchorfree/k8s-resource-updater
RUN curl https://glide.sh/get | sh \
    && cd /go/src/github.com/anchorfree/k8s-resource-updater \
    && glide install -v
RUN cd /go/src/github.com/anchorfree/k8s-resource-updater \
    && CGO_ENABLED=0 GOOS=linux go build -o /build/k8s-resource-updater main.go

FROM alpine

COPY --from=0 /build/k8s-resource-updater /

ENV CONSUL_TEMPLATE_VERSION 0.19.4

RUN apk add --update-cache --no-cache ca-certificates curl netcat-openbsd
RUN mkdir -p /opt/templates && mkdir /etc/consul-templater
RUN mkdir /output && wget -qO- https://releases.hashicorp.com/consul-template/${CONSUL_TEMPLATE_VERSION}/consul-template_${CONSUL_TEMPLATE_VERSION}_linux_amd64.tgz | tar xvz -C /bin

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
