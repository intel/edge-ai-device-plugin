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
	"time"
	"os"
	"os/exec"
	"strings"

	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/debug"
	dpapi "github.com/intel/intel-device-plugins-for-kubernetes/pkg/deviceplugin"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	// Device plugin settings.
	namespace  = "edgeai.intel.com"

	xlinkDevNode  = "/dev/xlnk"

	hddlAlive     = "hddlunite_service_alive.mutex"
	hddlReady     = "hddlunite_service_ready.mutex"
	hddlStartExit = "hddlunite_service_start_exit.mutex"
	hddlSocket    = "hddlunite_service.sock"
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

	// check if xlink dev node is present
	if !fileExists(dp.xlinkDev) {
		return devTree, nil
	}

	debug.Printf("xlink device node is present")

	nodes := []pluginapi.DeviceSpec{
		{
			HostPath:      xlinkDevNode,
			ContainerPath: xlinkDevNode,
			Permissions:   "rw",
		},
	}

	mounts := []pluginapi.Mount{
		{
			HostPath:      "/tmp/" + hddlAlive,
			ContainerPath: "/var/tmp/" + hddlAlive,
		},
		{
			HostPath:      "/tmp/" + hddlReady,
			ContainerPath: "/var/tmp/" + hddlReady,
		},
		{
			HostPath:      "/tmp/" + hddlStartExit,
			ContainerPath: "/var/tmp/" + hddlStartExit,
		},
		{
			HostPath:      "/tmp/" + hddlSocket,
			ContainerPath: "/var/tmp/" + hddlSocket,
		},
	}

	cmdout, _ := exec.Command("lspci", "-d", "8086:6240").Output()
	ss := strings.Split(string(cmdout), "6240")
	debug.Printf("detect %d kmb devices", len(ss) - 1)

	for i := 0; i < len(ss) - 1; i++ {
	        s := fmt.Sprintf("kmb-device-%d", i)
		devTree.AddDevice("kmb", s, dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
	}

	cmdout, _ = exec.Command("lspci", "-d", "8086:4fc0").Output()
	ss = strings.Split(string(cmdout), "4fc0")
	cmdout, _ = exec.Command("lspci", "-d", "8086:4fc1").Output()
	ss2 := strings.Split(string(cmdout), "4fc1")
	debug.Printf("detect %d thb devices", len(ss)/2 + len(ss2)/2)

	for i := 0; i < len(ss)/2 + len(ss2)/2; i++ {
	        s := fmt.Sprintf("thb-device-%d", i)
	        devTree.AddDevice("thb", s, dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
	}


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
