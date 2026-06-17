// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package container // import "go.opentelemetry.io/contrib/detectors/container"

import (
	"context"
	"errors"
	"os"
	"regexp"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const (
	defaultCgroupPath    = "/proc/self/cgroup"
	defaultMountInfoPath = "/proc/self/mountinfo"
)

var (
	empty                      = resource.Empty()
	errCannotReadContainerName = errors.New("failed to read hostname")
	// matches a 64 character hexadecimal container ID bounded by a
	// non-hexadecimal character so runtime prefixes and suffixes are excluded.
	containerIDPattern = regexp.MustCompile(`(?:^|[^0-9a-f])([0-9a-f]{64})(?:[^0-9a-f]|$)`)
)

// Create interface for methods needing to be mocked.
type detectorUtils interface {
	getContainerName() (string, error)
	getContainerID() (string, error)
}

// struct implements detectorUtils interface.
type containerDetectorUtils struct{}

// ResourceDetector collects resource information of the container the process is running in.
type ResourceDetector struct {
	utils detectorUtils
}

// compile time assertion that containerDetectorUtils implements detectorUtils interface.
var _ detectorUtils = (*containerDetectorUtils)(nil)

// compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// New returns a resource detector that will detect container resources.
func New() *ResourceDetector {
	return &ResourceDetector{
		utils: containerDetectorUtils{},
	}
}

// Detect finds associated resources when running inside a container.
func (detector *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	hostName, err := detector.utils.getContainerName()
	if err != nil {
		return empty, err
	}
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return empty, err
	}
	if containerID == "" {
		return empty, nil
	}

	attributes := []attribute.KeyValue{
		semconv.ContainerName(hostName),
		semconv.ContainerID(containerID),
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

// returns container ID from the cgroup hierarchy, falling back to mountinfo for cgroup v2.
func (containerDetectorUtils) getContainerID() (string, error) {
	if runtime.GOOS != "linux" {
		// Cgroups are used only under Linux.
		return "", nil
	}

	for _, path := range []string{defaultCgroupPath, defaultMountInfoPath} {
		fileData, err := os.ReadFile(path)
		if err != nil {
			// File not found, e.g. windows or running outside of a container.
			continue
		}
		if id := getCgroupContainerID(fileData); id != "" {
			return id, nil
		}
	}
	return "", nil
}

// returns host name reported by the kernel.
func (containerDetectorUtils) getContainerName() (string, error) {
	hostName, err := os.Hostname()
	if err != nil {
		return "", errCannotReadContainerName
	}
	return hostName, nil
}

// returns the first container ID found across the lines of fileData.
func getCgroupContainerID(fileData []byte) string {
	for line := range strings.SplitSeq(string(fileData), "\n") {
		if m := containerIDPattern.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}
