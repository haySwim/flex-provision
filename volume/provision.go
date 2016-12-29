/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volume

import (

	"github.com/golang/glog"
	"os/exec"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/types"
	"github.com/kubernetes-incubator/nfs-provisioner/controller"
)

const (
	// Name of the file where an s3fsProvisioner will store its identity
	identityFile = "flex-provisioner.identity"

	// are we allowed to set this? else make up our own
	annCreatedBy = "kubernetes.io/createdby"
	createdBy    = "flex-dynamic-provisioner"

	// A PV annotation for the identity of the s3fsProvisioner that provisioned it
	annProvisionerId = "Provisioner_Id"


)

func NewFlexProvisioner(client kubernetes.Interface, execCommand string) controller.Provisioner {
	return newFlexProvisionerInternal(client, execCommand)
}

func newFlexProvisionerInternal(client kubernetes.Interface, execCommand string) *flexProvisioner {
	var identity types.UID

	provisioner := &flexProvisioner{
		client:       client,
		execCommand:	execCommand,
		identity:     identity,
	}

	return provisioner
}

type flexProvisioner struct {
	client kubernetes.Interface
	execCommand string
	identity types.UID
}

var _ controller.Provisioner = &flexProvisioner{}

// Provision creates a volume i.e. the storage asset and returns a PV object for
// the volume.
func (p *flexProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	err := p.createVolume(options)
	if err != nil {
		return nil, err
	}

	annotations := make(map[string]string)
	annotations[annCreatedBy] = createdBy

	annotations[annProvisionerId] = string(p.identity)
	/*
		This PV won't work since there's nothing backing it.  the flex script
		is in flex/flex/flex  (that many layers are required for the flex volume plugin)
	 */
	pv := &v1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name:        options.PVName,
			Labels:      map[string]string{},
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.Capacity,
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{

				FlexVolume: &v1.FlexVolumeSource{
					Driver: "flex",
					Options: map[string]string{
					},

					ReadOnly: false,
				},
			},
		},
	}

	return pv, nil
}

func (p *flexProvisioner) createVolume(volumeOptions controller.VolumeOptions) (error) {

	cmd := exec.Command(p.execCommand, "provision")
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Failed to create volume %s, output: %s, error: %s",  volumeOptions, output, err.Error())
		return err
	}

	return  nil

}
