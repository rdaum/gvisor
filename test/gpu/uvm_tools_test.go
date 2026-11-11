// Copyright 2024 The gVisor Authors.
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

// Package uvm_tools_test tests ioctls on the NVIDIA UVM tools device.
package uvm_tools_test

import (
	"context"
	"strings"
	"testing"

	"gvisor.dev/gvisor/pkg/test/dockerutil"
)

// TestUVMToolsPopulatePageable checks that UVM_POPULATE_PAGEABLE on
// /dev/nvidia-uvm-tools fails with EINVAL and that the container survives.
// The test does not depend on ATS or on the UVM mode.
func TestUVMToolsPopulatePageable(t *testing.T) {
	ctx := context.Background()
	// The program does not use CUDA, but its image requires CUDA 12.8.
	cudaVersion, err := dockerutil.MaxSuportedCUDAVersion(ctx, t)
	if err != nil {
		t.Fatalf("failed to get CUDA version: %v", err)
	}
	if !cudaVersion.IsAtLeast(dockerutil.MustParseCudaVersion("12.8")) {
		t.Skipf("CUDA version %s is not at least 12.8, skipping test", cudaVersion)
	}
	isGVisor, err := dockerutil.IsGVisorRuntime(ctx)
	if err != nil {
		t.Fatalf("failed to determine the runtime: %v", err)
	}
	// The NVIDIA container toolkit creates the device node with the compute
	// capability and has no profiling capability. nvproxy also requires its
	// profiling capability, which the runtime must allow.
	capabilitySets := []string{"compute", "compute,utility"}
	if isGVisor {
		capabilitySets = []string{"compute,profiling", "compute,utility,profiling"}
	}
	for _, capabilities := range capabilitySets {
		// Container names come from test names and cannot contain commas.
		t.Run(strings.ReplaceAll(capabilities, ",", "_"), func(t *testing.T) {
			c := dockerutil.MakeContainer(ctx, t)
			defer c.CleanUp(ctx)
			opts, err := dockerutil.GPURunOpts(dockerutil.SniffGPUOpts{
				Capabilities:         capabilities,
				DisableSnifferReason: "the program asserts the ioctl result itself",
			})
			if err != nil {
				t.Fatalf("failed to get GPU run options: %v", err)
			}
			opts.Image = "gpu/cuda-tests-12-8"
			out, err := c.Run(ctx, opts, "/uvm_tools_populate_pageable")
			t.Logf("UVM tools ioctl output: %s", out)
			if err != nil {
				t.Fatalf("UVM tools ioctl regression failed: %v", err)
			}
			if !strings.Contains(out, "PASS: ") {
				t.Fatalf("UVM tools ioctl regression did not report a pass")
			}
		})
	}
}
