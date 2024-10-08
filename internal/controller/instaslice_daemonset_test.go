/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	inferencev1alpha1 "github.com/openshift/instaslice-operator/api/v1alpha1"
	runtimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCleanUp(t *testing.T) {

	// Create a fake Kubernetes client
	s := scheme.Scheme
	_ = inferencev1alpha1.AddToScheme(s)
	fakeClient := runtimefake.NewClientBuilder().WithScheme(s).Build()

	reconciler := &InstaSliceDaemonsetReconciler{
		Client: fakeClient,
		Scheme: s,
	}

	instaslice := &inferencev1alpha1.Instaslice{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Spec: inferencev1alpha1.InstasliceSpec{
			Prepared: map[string]inferencev1alpha1.PreparedDetails{
				"mig-uuid-1": {
					PodUUID:  "pod-uid-1",
					Parent:   "GPU-1",
					Giinfoid: 1,
					Ciinfoid: 1,
				},
			},
			Allocations: map[string]inferencev1alpha1.AllocationDetails{
				"allocation-1": {
					PodUUID:   "pod-uid-1",
					PodName:   "pod-name-1",
					Namespace: "default",
				},
			},
		},
	}
	errCreatingInstaSlice := fakeClient.Create(context.Background(), instaslice)
	assert.NoError(t, errCreatingInstaSlice)

	// Create a fake Node resource
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Status: v1.NodeStatus{
			//TODO: add test cases for other resources
			Capacity: v1.ResourceList{
				v1.ResourceName(AppendToInstaSlicePrefix("uid-1")): resource.MustParse("1"),
			},
		},
	}
	errCreatingNode := fakeClient.Create(context.Background(), node)
	assert.NoError(t, errCreatingNode)

	// Set the NODE_NAME environment variable
	setEnvErr := os.Setenv("NODE_NAME", "node-1")
	assert.NoError(t, setEnvErr, "error setting setenv")
	defer func() {
		err := os.Unsetenv("NODE_NAME")
		assert.NoError(t, err)
	}()

	// Keeping it around as it may be needed later
	// Create a fake Pod resource
	// pod := &v1.Pod{
	//  ObjectMeta: metav1.ObjectMeta{
	//      UID:       "pod-uid-1",
	//      Name:      "pod-name-1",
	//      Namespace: "default",
	//  },
	// }

	var updatedInstaslice inferencev1alpha1.Instaslice
	err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "node-1"}, &updatedInstaslice)
	assert.NoError(t, err)

	var updatedNode v1.Node
	errVerifyingInstaSliceResource := fakeClient.Get(context.Background(), types.NamespacedName{Name: "node-1"}, &updatedNode)
	assert.NoError(t, errVerifyingInstaSliceResource)

	_, exists := updatedNode.Status.Capacity[v1.ResourceName(AppendToInstaSlicePrefix("uid-1"))]
	assert.False(t, exists, fmt.Sprintf("resource '%s' should be deleted from the node's capacity", AppendToInstaSlicePrefix("uid-1")))

}
