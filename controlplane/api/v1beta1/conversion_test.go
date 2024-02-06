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
package v1beta1

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	bootstrapv1beta1 "github.com/cluster-api-provider-k3s/cluster-api-k3s/bootstrap/api/v1beta1"
	cabp3v1 "github.com/cluster-api-provider-k3s/cluster-api-k3s/controlplane/api/v1beta2"
)

func TestFuzzyConversion(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(Succeed())
	g.Expect(cabp3v1.AddToScheme(scheme)).To(Succeed())

	t.Run("for KThreesControlPlane", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &cabp3v1.KThreesControlPlane{},
		Spoke:       &KThreesControlPlane{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{KThreesServerConfigFuzzFunc},
	}))
}

func KThreesServerConfigFuzzFunc(_ serializer.CodecFactory) []interface{} {
	return []interface{}{
		KThreesServerConfigFuzzer,
	}
}

func KThreesServerConfigFuzzer(in *bootstrapv1beta1.KThreesServerConfig, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	// This field have been removed in v1beta2, data is going to be lost.
	in.DisableExternalCloudProvider = false
}
