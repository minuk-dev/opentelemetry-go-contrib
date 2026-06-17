// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const testContainerID = "7be92808767a667f35c8505cbf40d14e931ef6db5b0210329cf193b15ba9d605"

// MockDetectorUtils mocks the functions that need to be mocked.
type MockDetectorUtils struct {
	mock.Mock
}

func (detectorUtils *MockDetectorUtils) getContainerName() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

// successfully returns a resource with container.name and container.id when
// the process is running in a container.
func TestDetect(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return(testContainerID, nil)

	attributes := []attribute.KeyValue{
		semconv.ContainerName("container-Name"),
		semconv.ContainerID(testContainerID),
	}
	expected := resource.NewWithAttributes(semconv.SchemaURL, attributes...)

	detector := &ResourceDetector{utils: detectorUtils}
	res, err := detector.Detect(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, expected, res, "Resource returned is incorrect")
}

// returns an empty resource when no container ID can be determined.
func TestDetectNoContainer(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	detectorUtils.On("getContainerName").Return("host-Name", nil)
	detectorUtils.On("getContainerID").Return("", nil)

	detector := &ResourceDetector{utils: detectorUtils}
	res, err := detector.Detect(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, 0, res.Len())
}

// propagates an error reading the container name.
func TestDetectContainerNameError(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	detectorUtils.On("getContainerName").Return("", errCannotReadContainerName)

	detector := &ResourceDetector{utils: detectorUtils}
	res, err := detector.Detect(t.Context())
	assert.ErrorIs(t, err, errCannotReadContainerName)
	assert.Equal(t, empty, res)
}

// propagates an error reading the container ID.
func TestDetectContainerIDError(t *testing.T) {
	detectorUtils := new(MockDetectorUtils)
	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("", errors.New("boom"))

	detector := &ResourceDetector{utils: detectorUtils}
	res, err := detector.Detect(t.Context())
	assert.Error(t, err)
	assert.Equal(t, empty, res)
}

func TestNewUsesDefaultUtils(t *testing.T) {
	d := New()
	_, ok := d.utils.(containerDetectorUtils)
	assert.True(t, ok, "New should use containerDetectorUtils")
}

// getCgroupContainerID extracts the container ID from representative cgroup v1
// and mountinfo (cgroup v2) contents.
func TestGetCgroupContainerID(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "empty",
			data: "",
			want: "",
		},
		{
			name: "docker cgroup v1",
			data: "12:pids:/docker/" + testContainerID + "\n" +
				"11:hugetlb:/docker/" + testContainerID + "\n" +
				"1:name=systemd:/docker/" + testContainerID,
			want: testContainerID,
		},
		{
			name: "kubernetes cgroup v1",
			data: "11:cpuset:/kubepods/besteffort/pod1234abcd-12ab-34cd-56ef-1234567890ab/" + testContainerID,
			want: testContainerID,
		},
		{
			name: "systemd scope with runtime prefix",
			data: "0::/system.slice/docker-" + testContainerID + ".scope",
			want: testContainerID,
		},
		{
			name: "cri-containerd scope",
			data: "0::/.../cri-containerd-" + testContainerID + ".scope",
			want: testContainerID,
		},
		{
			name: "mountinfo cgroup v2",
			data: "2659 2641 0:256 /var/lib/docker/containers/" + testContainerID +
				"/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/sda1 rw",
			want: testContainerID,
		},
		{
			name: "no container id",
			data: "0::/init.scope\n11:cpuset:/\n10:memory:/user.slice",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getCgroupContainerID([]byte(tt.data)))
		})
	}
}
