FROM golang:1.24.3-alpine as build_deps

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go build -ldflags '-w -s' -a -installsuffix cgo -o helm-values-schemas main.go

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /workspace/helm-values-schemas helm-values-schemas

EXPOSE 8080

ENTRYPOINT ["/helm-values-schemas"]
