# spot-evaluator

`spot-evaluator` is a Go CLI that inspects Kubernetes worker nodes and estimates Spot savings against On-Demand prices, including possible replacement instance types that can reduce hourly costs.

## Features

- Lists worker node groups by instance type and AZ.
- Fetches current Spot and On-Demand pricing from AWS.
- Calculates spot savings percentage for current worker types.
- Suggests compatible replacement EC2 types in the same AZ.
- Shows estimated savings in dollars per node per hour and per node-group per hour.

## Requirements

- Go 1.22+
- Access to a Kubernetes cluster (`~/.kube/config`)
- AWS credentials with permission to call:
  - `ec2:DescribeSpotPriceHistory`
  - `ec2:DescribeInstanceTypes`
  - `ec2:DescribeInstanceTypeOfferings`
  - `pricing:GetProducts`

## Quick Start

```bash
go mod tidy
go run ./cmd/spot-evaluator
```

Backward-compatible legacy path (kept to reduce branch merge conflicts):

```bash
go run ./cmd/server
```

## Review status in this environment

I cannot pull from `main` in this execution environment because no git remote is configured for this local repository clone. To sync locally, run:

```bash
git remote add origin <your-repo-url>   # only if missing
git fetch origin
git checkout main
git pull --ff-only origin main
git checkout work
git merge main
```

## Testing against your Kubernetes cluster

### 1) Verify cluster connectivity

```bash
kubectl config current-context
kubectl get nodes -o wide
```

You should see the worker nodes you expect. The app reads node labels for:
- `node.kubernetes.io/instance-type`
- `topology.kubernetes.io/zone`
- `topology.kubernetes.io/region`
- `eks.amazonaws.com/capacityType`

### 2) Verify AWS identity/region

```bash
aws sts get-caller-identity
aws configure list
```

If you use profiles, export one before running:

```bash
export AWS_PROFILE=<profile-name>
export AWS_REGION=<region>
```

### 3) Install dependencies and run tests

```bash
go mod tidy
go test ./...
```

If your corporate network blocks `proxy.golang.org`, configure an internal proxy:

```bash
go env -w GOPROXY=https://<your-internal-go-proxy>,direct
```

### 4) Run the evaluator

```bash
go run ./cmd/spot-evaluator
```

Backward-compatible legacy path (kept to reduce branch merge conflicts):

```bash
go run ./cmd/server
```

Expected output sections:
- current node-group pricing table (instance, AZ, count, on-demand, spot, savings %)
- replacement recommendations with:
  - replacement instance type
  - replacement spot price
  - savings per node per hour
  - savings per group per hour

### 5) Optional: build a binary and run in CI/staging

```bash
go build -o bin/spot-evaluator ./cmd/spot-evaluator
./bin/spot-evaluator

# legacy compatible build path
go build -o bin/server ./cmd/server
```

### 6) Troubleshooting checklist

- **`missing go.sum entry`**: run `go mod tidy` where dependency egress is allowed.
- **Kubernetes auth errors**: check `KUBECONFIG` and current context.
- **AWS AccessDenied**: validate IAM permissions listed above.
- **No spot results for AZ/type**: verify the worker instance type and AZ are valid in your account/region.

## Project Layout

- `cmd/spot-evaluator/main.go` - primary CLI entrypoint for the `spot-evaluator` tool.
- `cmd/server/main.go` - legacy-compatible entrypoint retained to reduce merge conflicts with older branches.
- `internal/collector/` - Kubernetes node inventory collection.
- `internal/pricing/` - AWS Spot/On-Demand pricing and replacement logic.
- `pkg/models/` - Shared data models.

## Open Source

This repository is licensed under the MIT License.
See [LICENSE](./LICENSE).
