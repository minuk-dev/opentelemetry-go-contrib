# Container Resource detector

<!--[![PkgGoDev](https://pkg.go.dev/badge/go.opentelemetry.io/contrib/detectors/container)](https://pkg.go.dev/go.opentelemetry.io/contrib/detectors/container)-->

This package detects the [`container.id`](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/container.md)
resource attribute of the process it is running in by reading the container ID
from the process control group (cgroup).

Detection is only supported on Linux. On other platforms, or when the process
is not running in a container, the detector returns an empty resource without
an error so it composes cleanly with other detectors.
