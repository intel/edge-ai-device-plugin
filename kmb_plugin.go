// Copyright 2020 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/debug"
	dpapi "github.com/intel/intel-device-plugins-for-kubernetes/pkg/deviceplugin"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	// Device plugin settings.
	namespace  = "kmb.intel.com"

	xlinkDevNode  = "/dev/xlnk"
	vpusmmDevNode = "/dev/vpusmm0"
	driDevNode    = "/dev/dri/renderD129"
)

var (
	isdebug = flag.Int("debug", 1, "debug level (0..1)")
)

type devicePlugin struct {
	xlinkDev string
}

func newDevicePlugin(xlinkDev string) *devicePlugin {
	return &devicePlugin{
		xlinkDev:   xlinkDev,
	}
}

func (dp *devicePlugin) Scan(notifier dpapi.Notifier) error {
	for {
		devTree, err := dp.scan()
		if err != nil {
			return err
		}

		notifier.Notify(devTree)

		time.Sleep(5 * time.Second)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err == nil && info != nil {
		return !info.IsDir()
	}
	// regard all other case as abnormal
	return false
}

func (dp *devicePlugin) scan() (dpapi.DeviceTree, error) {
	fmt.Println("KMB device scanning started")

	devTree := dpapi.NewDeviceTree()

	if !fileExists(dp.xlinkDev) {
		return devTree, nil
	}

	if !fileExists(vpusmmDevNode) {
		return devTree, nil
	}

	if !fileExists(driDevNode) {
		return devTree, nil
	}

	nodes := []pluginapi.DeviceSpec{
		{
			HostPath:      dp.xlinkDev,
			ContainerPath: dp.xlinkDev,
			Permissions:   "rw",
		},
		{
			HostPath:      vpusmmDevNode,
			ContainerPath: vpusmmDevNode,
			Permissions:   "rw",
		},
		{
			HostPath:      driDevNode,
			ContainerPath: driDevNode,
			Permissions:   "rw",
		},
	}

	mounts := []pluginapi.Mount { }

	devTree.AddDevice("vpu",   "kmb-vpu-0",   dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
	devTree.AddDevice("codec", "kmb-codec-0", dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
	devTree.AddDevice("codec", "kmb-codec-1", dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
	devTree.AddDevice("codec", "kmb-codec-2", dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))

	return devTree, nil
}

func main() {
	if *isdebug > 0 {
		debug.Activate()
		debug.Printf("isdebug is on")
	}

	fmt.Println("KMB device plugin started")

	plugin := newDevicePlugin(xlinkDevNode)
	manager := dpapi.NewManager(namespace, plugin)
	manager.Run()
}
