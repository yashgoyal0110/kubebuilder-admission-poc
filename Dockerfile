# Must match the go directive in go.mod. hack/verify-go-version.sh enforces that in CI.
FROM --platform=$BUILDPLATFORM golang:1.25.1 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the module files first so dependency download is cached independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -a -o manager ./cmd

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
