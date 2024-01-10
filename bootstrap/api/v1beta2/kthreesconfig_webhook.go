/*


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

package v1beta2

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// SetupWebhookWithManager will setup the webhooks for the KThreesControlPlane.
func (c *KThreesConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-controlplane-cluster-x-k8s-io-v1beta2-kthreesconfig,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=bootstrap.cluster.x-k8s.io,resources=kthreesconfig,versions=v1beta2,name=validation.kthreesconfig.bootstrap.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-controlplane-cluster-x-k8s-io-v1beta2-kthreesconfig,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=bootstrap.cluster.x-k8s.io,resources=kthreesconfig,versions=v1beta2,name=default.kthreesconfig.bootstrap.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &KThreesConfig{}
var _ webhook.Validator = &KThreesConfig{}

// ValidateCreate will do any extra validation when creating a KThreesControlPlane.
func (c *KThreesConfig) ValidateCreate() error {
	return nil
}

// ValidateUpdate will do any extra validation when updating a KThreesControlPlane.
func (c *KThreesConfig) ValidateUpdate(runtime.Object) error {
	return nil
}

// ValidateDelete allows you to add any extra validation when deleting.
func (c *KThreesConfig) ValidateDelete() error {
	return nil
}

// Default will set default values for the KThreesControlPlane.
func (c *KThreesConfig) Default() {
}
