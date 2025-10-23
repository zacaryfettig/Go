package main

import (
	containerservice "github.com/pulumi/pulumi-azure-native-sdk/containerservice/v20230102preview"
	keyvault "github.com/pulumi/pulumi-azure-native-sdk/keyvault"
	network "github.com/pulumi/pulumi-azure-native-sdk/network"
	resources "github.com/pulumi/pulumi-azure-native-sdk/resources/v2/v20241101"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var resourceGroupName string = "rg1"
var resourceGroupLocation string = "westus"

func main() {
	//Run resource creation
	pulumi.Run(func(ctx *pulumi.Context) error {

		//Call the resource group creation function
		rg, err := createResourceGroup(ctx, resourceGroupName, resourceGroupLocation)
		if err != nil {
			return err
		}

		// You can now use rg.Name, rg.ID, etc. for other resources
		ctx.Export("resourceGroupName", rg.Name)

		vnet, err := NewVNet(ctx, "myVNet", pulumi.String(resourceGroupLocation), rg.Name)
		if err != nil {
			return err
		}

		subnet, err := NewSubnet(ctx, "subnet1", vnet, "10.10.10.0/24")
		if err != nil {
			return err
		}
		ctx.Export("subnetID", subnet.ID())

		_, err = createAKS(ctx, subnet.ID())
		if err != nil {
			return err
		}

		tenantID := pulumi.String("<your-tenant-id>")
		subnetID := subnet.ID() // from your existing subnet

		_, err = createKeyVault(ctx, "vault90348503485", resourceGroupName, resourceGroupLocation, tenantID, subnetID)
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

func NewVNet(ctx *pulumi.Context, name string, location pulumi.StringInput, rgName pulumi.StringInput) (*VNet, error) {
	var vnet VNet

	args := pulumi.Map{
		"resourceGroupName": rgName,
		"location":          location,
		"addressSpace": pulumi.Map{
			"addressPrefixes": pulumi.StringArray{
				pulumi.String("10.0.0.0/16"),
			},
		},
	}

	err := ctx.RegisterResource("Microsoft.Network/virtualNetworks", name, args, &vnet)
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
func createAKS(ctx *pulumi.Context, subnetID pulumi.StringInput) (*containerservice.ManagedCluster, error) {
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

		// Enable Azure Disk CSI driver
		StorageProfile: &containerservice.ManagedClusterStorageProfileArgs{
			DiskCSIDriver: &containerservice.ManagedClusterStorageProfileDiskCSIDriverArgs{
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
				VnetSubnetID:       subnetID,
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
		Location: pulumi.String(resourceGroupLocation),
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

		Identity: &containerservice.ManagedClusterIdentityArgs{
			Type: containerservice.ResourceIdentityTypeSystemAssigned,
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

func createKeyVault(ctx *pulumi.Context, vaultName, resourceGroupName, location string, tenantID pulumi.StringInput, subnetID pulumi.StringInput) (*keyvault.Vault, error) {

	// Create Key Vault
	vault, err := keyvault.NewVault(ctx, vaultName, &keyvault.VaultArgs{
		ResourceGroupName: pulumi.String(resourceGroupName),
		VaultName:         pulumi.String(vaultName),
		Location:          pulumi.String(location),
		Properties: &keyvault.VaultPropertiesArgs{
			TenantId: tenantID,
			Sku: &keyvault.SkuArgs{
				Name:   keyvault.SkuNameStandard,
				Family: pulumi.String(keyvault.SkuFamilyA),
			},
			AccessPolicies: keyvault.AccessPolicyEntryArray{
				&keyvault.AccessPolicyEntryArgs{
					TenantId: tenantID,
					ObjectId: pulumi.String("<your-object-id>"),
					Permissions: &keyvault.PermissionsArgs{
						Keys: pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("create"),
							pulumi.String("delete"),
						},
						Secrets: pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("set"),
							pulumi.String("delete"),
						},
						Certificates: pulumi.StringArray{
							pulumi.String("get"),
							pulumi.String("list"),
							pulumi.String("create"),
							pulumi.String("delete"),
						},
					},
				},
			},
			EnabledForDeployment:         pulumi.Bool(true),
			EnabledForTemplateDeployment: pulumi.Bool(true),
			EnabledForDiskEncryption:     pulumi.Bool(true),
			NetworkAcls: &keyvault.NetworkRuleSetArgs{
				DefaultAction: pulumi.String("Deny"),
				Bypass:        pulumi.String("AzureServices"),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// Create Private Endpoint Key Vault
	_, err = network.NewPrivateEndpoint(ctx, vaultName+"-pe", &network.PrivateEndpointArgs{
		ResourceGroupName: pulumi.String(resourceGroupName),
		Location:          pulumi.String(location),
		Subnet: &network.SubnetTypeArgs{
			Id: subnetID,
		},

		PrivateLinkServiceConnections: network.PrivateLinkServiceConnectionArray{
			&network.PrivateLinkServiceConnectionArgs{
				Name:                 pulumi.String("kv-endpoint-connection"),
				PrivateLinkServiceId: vault.ID(),
				GroupIds: pulumi.StringArray{
					pulumi.String("vault"),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return vault, nil
}
