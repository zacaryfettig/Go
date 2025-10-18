package main

import (
	"github.com/pulumi/pulumi-azure-native-sdk"
	containerservice "github.com/pulumi/pulumi-azure-native-sdk/containerservice/v20230102preview"
	resources "github.com/pulumi/pulumi-azure-native-sdk/resources/v2/v20241101"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var resourceGroupName string = "rg1"
var resourceGroupLocation string = "westus"

func main() {
	//Run resource creation
	pulumi.Run(func(ctx *pulumi.Context) error {

		//Call the resource group creation function
		_, err := createResourceGroup(ctx, resourceGroupName, resourceGroupLocation)
		if err != nil {
			return err
		}

		// You can now use rg.Name, rg.ID, etc. for other resources
		//		ctx.Export("resourceGroupName", rg.Name)

		vnet, err := NewVNet(ctx, "myVNet", pulumi.String(resourceGroupLocation), rg)
		if err != nil {
			return err
		}

		subnet, err := NewSubnet(ctx, "subnet1", vnet, "10.10.10.0/24")
		if err != nil {
			return err
		}
		ctx.Export("subnetID", subnet.ID())

		cluster, err := createAKS(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

// Function for Creating Resource Group
func createResourceGroup(ctx *pulumi.Context, name string, location string) (*resources.ResourceGroup, error) {
	rg, err := resources.NewResourceGroup(ctx, name, &resources.ResourceGroupArgs{
		Location:          pulumi.String(location),
		ResourceGroupName: pulumi.String(name),
	})
	if err != nil {
		return nil, err
	}
	return rg, nil
}

type VNet struct {
	pulumi.CustomResourceState
}

func NewVNet(ctx *pulumi.Context, name string, location pulumi.StringInput, parent pulumi.Resource) (*VNet, error) {
	var vnet VNet

	args := pulumi.Map{
		"location": location,
		"addressSpace": pulumi.Map{
			"addressPrefixes": pulumi.StringArray{
				pulumi.String("10.0.0.0/16"),
			},
		},
	}

	err := ctx.RegisterResource("Microsoft.Network/virtualNetworks", name, args, &vnet, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	return &vnet, nil
}

type Subnet struct {
	pulumi.CustomResourceState
}

func NewSubnet(ctx *pulumi.Context, name string, vnet *VNet, prefix string) (*Subnet, error) {
	var subnet Subnet

	args := pulumi.Map{
		"addressPrefix": pulumi.String(prefix),
	}

	err := ctx.RegisterResource("Microsoft.Network/virtualNetworks/subnets", name, args, &subnet, pulumi.Parent(vnet))
	if err != nil {
		return nil, err
	}

	return &subnet, nil
}

// createAKS provisions an Azure Kubernetes Service (AKS) cluster
func createAKS(ctx *pulumi.Context) (*containerservice.ManagedCluster, error) {
	cluster, err := containerservice.NewManagedCluster(ctx, "managedCluster", &containerservice.ManagedClusterArgs{
		AddonProfiles: containerservice.ManagedClusterAddonProfileMap{
			"azureKeyvaultSecretsProvider": &containerservice.ManagedClusterAddonProfileArgs{
				Config: pulumi.StringMap{
					"enableSecretRotation": pulumi.String("true"),
					"rotationPollInterval": pulumi.String("2m"),
				},
				Enabled: pulumi.Bool(true),
			},
		},
		AgentPoolProfiles: containerservice.ManagedClusterAgentPoolProfileArray{
			&containerservice.ManagedClusterAgentPoolProfileArgs{
				Count:              pulumi.Int(2),
				EnableNodePublicIP: pulumi.Bool(false),
				Mode:               pulumi.String(containerservice.AgentPoolModeSystem),
				Name:               pulumi.String("nodepool1"),
				OsType:             pulumi.String(containerservice.OSTypeLinux),
				Type:               pulumi.String(containerservice.AgentPoolTypeVirtualMachineScaleSets),
				VmSize:             pulumi.String("Standard_DS2_v2"),
			},
		},
		AutoScalerProfile: &containerservice.ManagedClusterPropertiesAutoScalerProfileArgs{
			ScaleDownDelayAfterAdd: pulumi.String("15m"),
			ScanInterval:           pulumi.String("20s"),
		},
		DiskEncryptionSetID:     pulumi.String("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg1/providers/Microsoft.Compute/diskEncryptionSets/des"),
		DnsPrefix:               pulumi.String("dnsprefix1"),
		EnablePodSecurityPolicy: pulumi.Bool(true),
		EnableRBAC:              pulumi.Bool(true),
		KubernetesVersion:       pulumi.String(""),
		LinuxProfile: &containerservice.ContainerServiceLinuxProfileArgs{
			AdminUsername: pulumi.String("azureuser"),
			Ssh: &containerservice.ContainerServiceSshConfigurationArgs{
				PublicKeys: containerservice.ContainerServiceSshPublicKeyArray{
					&containerservice.ContainerServiceSshPublicKeyArgs{
						KeyData: pulumi.String("keydata"),
					},
				},
			},
		},
		Location: pulumi.String("location1"),
		NetworkProfile: &containerservice.ContainerServiceNetworkProfileArgs{
			LoadBalancerProfile: &containerservice.ManagedClusterLoadBalancerProfileArgs{
				ManagedOutboundIPs: &containerservice.ManagedClusterLoadBalancerProfileManagedOutboundIPsArgs{
					Count: pulumi.Int(2),
				},
			},
			LoadBalancerSku: pulumi.String(containerservice.LoadBalancerSkuStandard),
			OutboundType:    pulumi.String(containerservice.OutboundTypeLoadBalancer),
		},
		ResourceGroupName: pulumi.String("rg1"),
		ResourceName:      pulumi.String("clustername1"),
		ServicePrincipalProfile: &containerservice.ManagedClusterServicePrincipalProfileArgs{
			ClientId: pulumi.String("clientid"),
			Secret:   pulumi.String("secret"),
		},
		Sku: &containerservice.ManagedClusterSKUArgs{
			Name: pulumi.String("Basic"),
			Tier: pulumi.String(containerservice.ManagedClusterSKUTierFree),
		},
		Tags: pulumi.StringMap{
			"archv2": pulumi.String(""),
			"tier":   pulumi.String("production"),
		},
		WindowsProfile: &containerservice.ManagedClusterWindowsProfileArgs{
			AdminPassword: pulumi.String("replacePassword1234$"),
			AdminUsername: pulumi.String("azureuser"),
		},
	})
	if err != nil {
		return nil, err
	}

	return cluster, nil
}
