//go:build acceptance || all

//testacc:tier=heavy
//testacc:resource=vm

/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResourceVMDiskFormatFromBlockStorage verifies that a VM disk on a
// block-backed storage, created without an explicit file_format, reads back as
// "raw" and produces a stable (empty) plan on refresh.
//
// Proxmox omits format= from the VM config when it equals the storage default,
// which is always the case on block storage (raw), so the provider recovers it
// from the storage on refresh. A full-privilege token (as used here) reads the
// authoritative content endpoint; a read-only token lacking VM.Config.Disk on
// the owning VM instead derives the format from the volume ID. This test runs
// the former path and guards the resolved format and a stable (no-diff) plan.
func TestAccResourceVMDiskFormatFromBlockStorage(t *testing.T) {
	t.Parallel()

	te := InitEnvironment(t)

	resourceName := "proxmox_virtual_environment_vm.test_disk_format"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: te.AccProviders,
		Steps: []resource.TestStep{
			{
				// local-lvm is block-backed (lvm-thin); no file_format is set, so
				// PVE stores raw as the storage default and omits format= from the
				// VM config. The provider must resolve "raw" on read.
				Config: te.RenderConfig(`
					resource "proxmox_virtual_environment_vm" "test_disk_format" {
						node_name = "{{.NodeName}}"
						started   = false
						name      = "test-disk-format"

						disk {
							datastore_id = "local-lvm"
							interface    = "virtio0"
							size         = 8
						}
					}`),
				Check: resource.ComposeTestCheckFunc(
					ResourceAttributes(resourceName, map[string]string{
						"disk.0.datastore_id": "local-lvm",
						"disk.0.file_format":  "raw",
					}),
				),
			},
			{
				// Re-applying the identical config must be a no-op: the format
				// resolved on refresh has to match what is in state, otherwise a
				// perpetual diff would surface here.
				Config: te.RenderConfig(`
					resource "proxmox_virtual_environment_vm" "test_disk_format" {
						node_name = "{{.NodeName}}"
						started   = false
						name      = "test-disk-format"

						disk {
							datastore_id = "local-lvm"
							interface    = "virtio0"
							size         = 8
						}
					}`),
				PlanOnly: true,
			},
		},
	})
}
