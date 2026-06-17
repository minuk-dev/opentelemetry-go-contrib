// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package container provides a [resource.Detector] which supports detecting
attributes specific to the container the process is running in.

The container ID is read from the process control group (cgroup), so detection
is only supported on Linux. When a container is detected, the following
attributes are added:

  - container.id
  - container.name

[container]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/container.md
*/
package container // import "go.opentelemetry.io/contrib/detectors/container"
