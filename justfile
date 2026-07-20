set quiet

binary := "k3d-router"

[private]
default:
    @just --list

[doc("Build the binary")]
build:
    go build -o {{binary}} .

[doc("Format code (goimports + gofumpt)")]
fmt:
    go tool goimports -w .
    go tool gofumpt -w .

[doc("Run go vet")]
vet:
    go vet ./...

[doc("Run the test suite")]
test:
    go test ./...

[doc("Run integration tests (needs Docker; k3d tier needs IT_K3D_CLUSTER + IT_K3D_HOSTNAME)")]
test-integration:
    go test -tags=integration ./internal/engine ./internal/router

[doc("Vet + test")]
check: vet test

[doc("Install to GOBIN / $GOPATH/bin")]
install:
    go install .

[doc("Run the CLI without building (e.g. just run up)")]
run *args:
    go run . {{args}}
