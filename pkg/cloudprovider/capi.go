package cloudprovider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/karpenter-provider-aws/pkg/apis/v1beta1"
	"github.com/aws/karpenter-provider-aws/pkg/providers/instancetype"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	corev1beta1 "sigs.k8s.io/karpenter/pkg/apis/v1beta1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/events"
	"sigs.k8s.io/karpenter/pkg/scheduling"
	"sigs.k8s.io/karpenter/pkg/utils/resources"
)

var _ cloudprovider.CloudProvider = (*CAPI)(nil)

type CAPI struct {
	instanceTypeProvider instancetype.Provider
	kubeClient           client.Client
	recorder             events.Recorder
}

func CAPINew(instanceTypeProvider instancetype.Provider, recorder events.Recorder,
	kubeClient client.Client) *CAPI {
	return &CAPI{
		instanceTypeProvider: instanceTypeProvider,
		kubeClient:           kubeClient,
		recorder:             recorder,
	}
}

// GetSupportedNodeClass returns the group, version, and kind of the CloudProvider NodeClass
func (capi *CAPI) GetSupportedNodeClasses() []schema.GroupVersionKind {
	return nil
}

func (c *CAPI) LivenessProbe(req *http.Request) error {
	return c.instanceTypeProvider.LivenessProbe(req)
}

func (c *CAPI) Get(ctx context.Context, providerID string) (*corev1beta1.NodeClaim, error) {
	// Implement the Get method logic here
	return nil, nil
}

func (c *CAPI) Create(ctx context.Context, nodeClaim *corev1beta1.NodeClaim) (*corev1beta1.NodeClaim, error) {
	// DESIGN: either have a capiInfraMachine <-> ec2 nodeClass conversion
	// or/and modify c.instanceTypeProvider.List signature specific values instead of a class.
	instanceTypes, err := c.resolveInstanceTypes(ctx, nodeClaim, &v1beta1.EC2NodeClass{})
	if err != nil {
		return nil, fmt.Errorf("resolving instance types, %w", err)
	}
	if len(instanceTypes) == 0 {
		return nil, cloudprovider.NewInsufficientCapacityError(fmt.Errorf("all requested instance types were unavailable during launch"))
	}

	// DESIGN: Delegation on CAPI:
	// Fetch NodeClassRef.
	// awsMachinePoolTemplate could be passed there. Otherwise convert to provider specific infraTemplate if needed.
	// Create MachinePool/MachineDeployment with 1 replica referencing template above.
	// Let CAPA make a createFleet request to AWS.
	return nil, nil
}

func (c *CAPI) Delete(ctx context.Context, nodeClaim *corev1beta1.NodeClaim) error {
	return nil
}

func (c *CAPI) List(ctx context.Context) ([]*corev1beta1.NodeClaim, error) {
	// Implement the List method logic here
	return nil, nil
}

func (c *CAPI) GetInstanceTypes(ctx context.Context, nodePool *corev1beta1.NodePool) ([]*cloudprovider.InstanceType, error) {
	// Implement the GetInstanceTypes method logic here
	return nil, nil
}

func (c *CAPI) IsDrifted(context.Context, *corev1beta1.NodeClaim) (cloudprovider.DriftReason, error) {
	// Implement the IsDrifted method logic here
	return "", nil
}

func (c *CAPI) Name() string {
	// Implement the Name method logic here
	return ""
}

// Filter out instance types that don't meet the requirements
func (c *CAPI) resolveInstanceTypes(ctx context.Context, nodeClaim *corev1beta1.NodeClaim, nodeClass *v1beta1.EC2NodeClass) ([]*cloudprovider.InstanceType, error) {
	instanceTypes, err := c.instanceTypeProvider.List(ctx, nodeClaim.Spec.Kubelet, nodeClass)
	if err != nil {
		return nil, fmt.Errorf("getting instance types, %w", err)
	}
	reqs := scheduling.NewNodeSelectorRequirementsWithMinValues(nodeClaim.Spec.Requirements...)
	return lo.Filter(instanceTypes, func(i *cloudprovider.InstanceType, _ int) bool {
		return reqs.Compatible(i.Requirements, scheduling.AllowUndefinedWellKnownLabels) == nil &&
			len(i.Offerings.Compatible(reqs).Available()) > 0 &&
			resources.Fits(nodeClaim.Spec.Resources.Requests, i.Allocatable())
	}), nil
}


This illustrates a possible design forward for a CAPI provider:
- Preserve the benefits of CAPI cloud agnosticity to manage indivdual instances lifecycle.
- Reuse as much as possible from Karpenter provider native controllers.
