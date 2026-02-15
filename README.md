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
go run ./cmd/server
```

## Testing

```bash
go test ./...
```

If your environment blocks module downloads, set an accessible GOPROXY or run tests in an environment with internet access to fetch dependencies.

## Project Layout

- `cmd/server/main.go` - CLI entrypoint.
- `internal/collector/` - Kubernetes node inventory collection.
- `internal/pricing/` - AWS Spot/On-Demand pricing and replacement logic.
- `pkg/models/` - Shared data models.

## Open Source

This repository is licensed under the MIT License.
See [LICENSE](./LICENSE).
