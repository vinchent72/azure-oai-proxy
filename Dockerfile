FROM golang:1.25.5 AS builder
WORKDIR /build
ARG TARGETOS=linux
ARG TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN if [ -n "$TARGETARCH" ]; then \
		CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o azure-oai-proxy . ; \
	else \
		CGO_ENABLED=0 GOOS=$TARGETOS go build -trimpath -ldflags="-s -w" -o azure-oai-proxy . ; \
	fi

FROM gcr.io/distroless/base-debian12
COPY --from=builder /build/azure-oai-proxy /
EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]