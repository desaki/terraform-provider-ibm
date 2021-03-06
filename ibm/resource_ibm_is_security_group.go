package ibm

import (
	"fmt"
	"reflect"

	"github.com/IBM/vpc-go-sdk/vpcclassicv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	isSecurityGroupName          = "name"
	isSecurityGroupVPC           = "vpc"
	isSecurityGroupRules         = "rules"
	isSecurityGroupResourceGroup = "resource_group"
)

func resourceIBMISSecurityGroup() *schema.Resource {

	return &schema.Resource{
		Create:   resourceIBMISSecurityGroupCreate,
		Read:     resourceIBMISSecurityGroupRead,
		Update:   resourceIBMISSecurityGroupUpdate,
		Delete:   resourceIBMISSecurityGroupDelete,
		Exists:   resourceIBMISSecurityGroupExists,
		Importer: &schema.ResourceImporter{},

		Schema: map[string]*schema.Schema{

			isSecurityGroupName: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "Security group name",
				ValidateFunc: validateISName,
			},
			isSecurityGroupVPC: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Security group's resource group id",
				ForceNew:    true,
			},

			isSecurityGroupRules: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Security Rules",
				Elem: &schema.Resource{
					Schema: makeIBMISSecurityRuleSchema(),
				},
			},

			isSecurityGroupResourceGroup: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Resource Group ID",
			},

			ResourceControllerURL: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The URL of the IBM Cloud dashboard that can be used to explore and view details about this instance",
			},

			ResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the resource",
			},

			ResourceCRN: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The crn of the resource",
			},

			ResourceGroupName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The resource group name in which resource is provisioned",
			},
		},
	}
}

func resourceIBMISSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	userDetails, err := meta.(ClientSession).BluemixUserDetails()
	if err != nil {
		return err
	}
	vpc := d.Get(isSecurityGroupVPC).(string)
	if userDetails.generation == 1 {
		err := classicSgCreate(d, meta, vpc)
		if err != nil {
			return err
		}
	} else {
		err := sgCreate(d, meta, vpc)
		if err != nil {
			return err
		}
	}
	return resourceIBMISSecurityGroupRead(d, meta)
}

func classicSgCreate(d *schema.ResourceData, meta interface{}, vpc string) error {
	sess, err := classicVpcClient(meta)
	if err != nil {
		return err
	}
	createSecurityGroupOptions := &vpcclassicv1.CreateSecurityGroupOptions{
		VPC: &vpcclassicv1.VPCIdentity{
			ID: &vpc,
		},
	}
	var rg, name string
	if grp, ok := d.GetOk(isSecurityGroupResourceGroup); ok {
		rg = grp.(string)
		createSecurityGroupOptions.ResourceGroup = &vpcclassicv1.ResourceGroupIdentity{
			ID: &rg,
		}
	}
	if nm, ok := d.GetOk(isSecurityGroupName); ok {
		name = nm.(string)
		createSecurityGroupOptions.Name = &name
	}

	sg, response, err := sess.CreateSecurityGroup(createSecurityGroupOptions)
	if err != nil {
		return fmt.Errorf("Error while creating Security Group %s\n%s", err, response)
	}
	d.SetId(*sg.ID)
	return nil
}

func sgCreate(d *schema.ResourceData, meta interface{}, vpc string) error {
	sess, err := vpcClient(meta)
	if err != nil {
		return err
	}
	createSecurityGroupOptions := &vpcv1.CreateSecurityGroupOptions{
		VPC: &vpcv1.VPCIdentity{
			ID: &vpc,
		},
	}
	var rg, name string
	if grp, ok := d.GetOk(isSecurityGroupResourceGroup); ok {
		rg = grp.(string)
		createSecurityGroupOptions.ResourceGroup = &vpcv1.ResourceGroupIdentity{
			ID: &rg,
		}
	}
	if nm, ok := d.GetOk(isSecurityGroupName); ok {
		name = nm.(string)
		createSecurityGroupOptions.Name = &name
	}
	sg, response, err := sess.CreateSecurityGroup(createSecurityGroupOptions)
	if err != nil {
		return fmt.Errorf("Error while creating Security Group %s\n%s", err, response)
	}
	d.SetId(*sg.ID)
	return nil
}

func resourceIBMISSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	userDetails, err := meta.(ClientSession).BluemixUserDetails()
	if err != nil {
		return err
	}
	id := d.Id()
	if userDetails.generation == 1 {
		err := classicSgGet(d, meta, id)
		if err != nil {
			return err
		}
	} else {
		err := sgGet(d, meta, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func classicSgGet(d *schema.ResourceData, meta interface{}, id string) error {
	sess, err := classicVpcClient(meta)
	if err != nil {
		return err
	}
	getSecurityGroupOptions := &vpcclassicv1.GetSecurityGroupOptions{
		ID: &id,
	}
	group, response, err := sess.GetSecurityGroup(getSecurityGroupOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error getting Security Group : %s\n%s", err, response)
	}
	d.Set(isSecurityGroupName, *group.Name)
	d.Set(isSecurityGroupVPC, *group.VPC.ID)
	rules := make([]map[string]interface{}, 0)
	if len(group.Rules) > 0 {
		for _, rule := range group.Rules {
			switch reflect.TypeOf(rule).String() {
			case "*vpcclassicv1.SecurityGroupRuleProtocolIcmp":
				{
					rule := rule.(*vpcclassicv1.SecurityGroupRuleProtocolIcmp)
					r := make(map[string]interface{})
					if rule.Code != nil {
						r[isSecurityGroupRuleCode] = int(*rule.Code)
					}
					if rule.Type != nil {
						r[isSecurityGroupRuleType] = int(*rule.Type)
					}
					r[isSecurityGroupRuleDirection] = *rule.Direction
					r[isSecurityGroupRuleIPVersion] = *rule.IPVersion
					if rule.Protocol != nil {
						r[isSecurityGroupRuleProtocol] = *rule.Protocol
					}
					if rule.Remote != nil && reflect.ValueOf(rule.Remote).IsNil() == false {
						for k, v := range rule.Remote.(map[string]interface{}) {
							if k == "id" || k == "address" || k == "cidr_block" {
								r[isSecurityGroupRuleRemote] = v.(string)
								break
							}
						}
					}
					rules = append(rules, r)
				}
			case "*vpcclassicv1.SecurityGroupRuleProtocolAll":
				{
					rule := rule.(*vpcclassicv1.SecurityGroupRuleProtocolAll)
					r := make(map[string]interface{})
					r[isSecurityGroupRuleDirection] = *rule.Direction
					r[isSecurityGroupRuleIPVersion] = *rule.IPVersion
					if rule.Protocol != nil {
						r[isSecurityGroupRuleProtocol] = *rule.Protocol
					}
					if rule.Remote != nil && reflect.ValueOf(rule.Remote).IsNil() == false {
						for k, v := range rule.Remote.(map[string]interface{}) {
							if k == "id" || k == "address" || k == "cidr_block" {
								r[isSecurityGroupRuleRemote] = v.(string)
								break
							}
						}
					}
					rules = append(rules, r)
				}
			case "*vpcclassicv1.SecurityGroupRuleProtocolTcpudp":
				{
					rule := rule.(*vpcclassicv1.SecurityGroupRuleProtocolTcpudp)
					r := make(map[string]interface{})
					if rule.PortMin != nil {
						r[isSecurityGroupRulePortMin] = int(*rule.PortMin)
					}
					if rule.PortMax != nil {
						r[isSecurityGroupRulePortMax] = int(*rule.PortMax)
					}
					r[isSecurityGroupRuleDirection] = *rule.Direction
					r[isSecurityGroupRuleIPVersion] = *rule.IPVersion
					if rule.Protocol != nil {
						r[isSecurityGroupRuleProtocol] = *rule.Protocol
					}
					if rule.Remote != nil && reflect.ValueOf(rule.Remote).IsNil() == false {
						for k, v := range rule.Remote.(map[string]interface{}) {
							if k == "id" || k == "address" || k == "cidr_block" {
								r[isSecurityGroupRuleRemote] = v.(string)
								break
							}
						}
					}
					rules = append(rules, r)
				}
			}
		}
	}
	d.Set(isSecurityGroupRules, rules)
	d.SetId(*group.ID)
	if group.ResourceGroup != nil {
		d.Set(isSecurityGroupResourceGroup, group.ResourceGroup.ID)
		rsMangClient, err := meta.(ClientSession).ResourceManagementAPIv2()
		if err != nil {
			return err
		}
		grp, err := rsMangClient.ResourceGroup().Get(*group.ResourceGroup.ID)
		if err != nil {
			return err
		}
		d.Set(ResourceGroupName, grp.Name)
	}
	controller, err := getBaseController(meta)
	if err != nil {
		return err
	}
	d.Set(ResourceControllerURL, controller+"/vpc/network/securityGroups")
	d.Set(ResourceName, *group.Name)
	d.Set(ResourceCRN, *group.CRN)
	return nil
}

func sgGet(d *schema.ResourceData, meta interface{}, id string) error {
	sess, err := vpcClient(meta)
	if err != nil {
		return err
	}
	getSecurityGroupOptions := &vpcv1.GetSecurityGroupOptions{
		ID: &id,
	}
	group, response, err := sess.GetSecurityGroup(getSecurityGroupOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error getting Security Group : %s\n%s", err, response)
	}
	d.Set(isSecurityGroupName, *group.Name)
	d.Set(isSecurityGroupVPC, *group.VPC.ID)
	rules := make([]map[string]interface{}, 0)
	if len(group.Rules) > 0 {
		for _, rule := range group.Rules {
			switch reflect.TypeOf(rule).String() {
			case "*vpcv1.SecurityGroupRuleProtocolIcmp":
				{
					rule := rule.(*vpcv1.SecurityGroupRuleProtocolIcmp)
					r := make(map[string]interface{})
					if rule.Code != nil {
						r[isSecurityGroupRuleCode] = int(*rule.Code)
					}
					if rule.Type != nil {
						r[isSecurityGroupRuleType] = int(*rule.Type)
					}
					r[isSecurityGroupRuleDirection] = *rule.Direction
					r[isSecurityGroupRuleIPVersion] = *rule.IPVersion
					if rule.Protocol != nil {
						r[isSecurityGroupRuleProtocol] = *rule.Protocol
					}
					if rule.Remote != nil && reflect.ValueOf(rule.Remote).IsNil() == false {
						for k, v := range rule.Remote.(map[string]interface{}) {
							if k == "id" || k == "address" || k == "cidr_block" {
								r[isSecurityGroupRuleRemote] = v.(string)
								break
							}
						}
					}
					rules = append(rules, r)
				}
			case "*vpcv1.SecurityGroupRuleProtocolAll":
				{
					rule := rule.(*vpcv1.SecurityGroupRuleProtocolAll)
					r := make(map[string]interface{})
					r[isSecurityGroupRuleDirection] = *rule.Direction
					r[isSecurityGroupRuleIPVersion] = *rule.IPVersion
					if rule.Protocol != nil {
						r[isSecurityGroupRuleProtocol] = *rule.Protocol
					}
					if rule.Remote != nil && reflect.ValueOf(rule.Remote).IsNil() == false {
						for k, v := range rule.Remote.(map[string]interface{}) {
							if k == "id" || k == "address" || k == "cidr_block" {
								r[isSecurityGroupRuleRemote] = v.(string)
								break
							}
						}
					}
					rules = append(rules, r)
				}
			case "*vpcv1.SecurityGroupRuleProtocolTcpudp":
				{
					rule := rule.(*vpcv1.SecurityGroupRuleProtocolTcpudp)
					r := make(map[string]interface{})
					if rule.PortMin != nil {
						r[isSecurityGroupRulePortMin] = int(*rule.PortMin)
					}
					if rule.PortMax != nil {
						r[isSecurityGroupRulePortMax] = int(*rule.PortMax)
					}
					r[isSecurityGroupRuleDirection] = *rule.Direction
					r[isSecurityGroupRuleIPVersion] = *rule.IPVersion
					if rule.Protocol != nil {
						r[isSecurityGroupRuleProtocol] = *rule.Protocol
					}
					if rule.Remote != nil && reflect.ValueOf(rule.Remote).IsNil() == false {
						for k, v := range rule.Remote.(map[string]interface{}) {
							if k == "id" || k == "address" || k == "cidr_block" {
								r[isSecurityGroupRuleRemote] = v.(string)
								break
							}
						}
					}
					rules = append(rules, r)
				}
			}
		}
	}
	d.Set(isSecurityGroupRules, rules)
	d.SetId(*group.ID)
	if group.ResourceGroup != nil {
		d.Set(isSecurityGroupResourceGroup, group.ResourceGroup.ID)
		d.Set(ResourceGroupName, group.ResourceGroup.Name)
	}
	controller, err := getBaseController(meta)
	if err != nil {
		return err
	}
	d.Set(ResourceControllerURL, controller+"/vpc-ext/network/securityGroups")
	d.Set(ResourceName, *group.Name)
	d.Set(ResourceCRN, *group.CRN)
	return nil
}

func resourceIBMISSecurityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	userDetails, err := meta.(ClientSession).BluemixUserDetails()
	if err != nil {
		return err
	}
	id := d.Id()
	name := ""
	hasChanged := false

	if d.HasChange(isSecurityGroupName) {
		name = d.Get(isSecurityGroupName).(string)
		hasChanged = true
	} else {
		return resourceIBMISSecurityGroupRead(d, meta)
	}
	if userDetails.generation == 1 {
		err := classicSgUpdate(d, meta, id, name, hasChanged)
		if err != nil {
			return err
		}
	} else {
		err := sgUpdate(d, meta, id, name, hasChanged)
		if err != nil {
			return err
		}
	}
	return resourceIBMISSecurityGroupRead(d, meta)
}

func classicSgUpdate(d *schema.ResourceData, meta interface{}, id, name string, hasChanged bool) error {
	sess, err := classicVpcClient(meta)
	if err != nil {
		return err
	}
	if hasChanged {
		updateSecurityGroupOptions := &vpcclassicv1.UpdateSecurityGroupOptions{
			ID:   &id,
			Name: &name,
		}
		_, response, err := sess.UpdateSecurityGroup(updateSecurityGroupOptions)
		if err != nil {
			return fmt.Errorf("Error Updating Security Group : %s\n%s", err, response)
		}
	}
	return nil
}

func sgUpdate(d *schema.ResourceData, meta interface{}, id, name string, hasChanged bool) error {
	sess, err := vpcClient(meta)
	if err != nil {
		return err
	}
	if hasChanged {
		updateSecurityGroupOptions := &vpcv1.UpdateSecurityGroupOptions{
			ID:   &id,
			Name: &name,
		}
		_, response, err := sess.UpdateSecurityGroup(updateSecurityGroupOptions)
		if err != nil {
			return fmt.Errorf("Error Updating Security Group : %s\n%s", err, response)
		}
	}
	return nil
}

func resourceIBMISSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	userDetails, err := meta.(ClientSession).BluemixUserDetails()
	if err != nil {
		return err
	}
	id := d.Id()
	if userDetails.generation == 1 {
		err := classicSgDelete(d, meta, id)
		if err != nil {
			return err
		}
	} else {
		err := sgDelete(d, meta, id)
		if err != nil {
			return err
		}
	}
	d.SetId("")
	return nil
}

func classicSgDelete(d *schema.ResourceData, meta interface{}, id string) error {
	sess, err := classicVpcClient(meta)
	if err != nil {
		return err
	}
	getSecurityGroupOptions := &vpcclassicv1.GetSecurityGroupOptions{
		ID: &id,
	}
	_, response, err := sess.GetSecurityGroup(getSecurityGroupOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Getting Security Group (%s): %s\n%s", id, err, response)
	}

	deleteSecurityGroupOptions := &vpcclassicv1.DeleteSecurityGroupOptions{
		ID: &id,
	}
	response, err = sess.DeleteSecurityGroup(deleteSecurityGroupOptions)
	if err != nil {
		return fmt.Errorf("Error Deleting Security Group : %s\n%s", err, response)
	}
	d.SetId("")
	return nil
}

func sgDelete(d *schema.ResourceData, meta interface{}, id string) error {
	sess, err := vpcClient(meta)
	if err != nil {
		return err
	}
	getSecurityGroupOptions := &vpcv1.GetSecurityGroupOptions{
		ID: &id,
	}
	_, response, err := sess.GetSecurityGroup(getSecurityGroupOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Getting Security Group (%s): %s\n%s", id, err, response)
	}

	deleteSecurityGroupOptions := &vpcv1.DeleteSecurityGroupOptions{
		ID: &id,
	}
	response, err = sess.DeleteSecurityGroup(deleteSecurityGroupOptions)
	if err != nil {
		return fmt.Errorf("Error Deleting Security Group : %s\n%s", err, response)
	}
	d.SetId("")
	return nil
}

func resourceIBMISSecurityGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	userDetails, err := meta.(ClientSession).BluemixUserDetails()
	if err != nil {
		return false, err
	}
	id := d.Id()
	if userDetails.generation == 1 {
		exists, err := classicSgExists(d, meta, id)
		return exists, err
	} else {
		exists, err := sgExists(d, meta, id)
		return exists, err
	}
}

func classicSgExists(d *schema.ResourceData, meta interface{}, id string) (bool, error) {
	sess, err := classicVpcClient(meta)
	if err != nil {
		return false, err
	}
	getSecurityGroupOptions := &vpcclassicv1.GetSecurityGroupOptions{
		ID: &id,
	}
	_, response, err := sess.GetSecurityGroup(getSecurityGroupOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("Error getting Security Group: %s\n%s", err, response)
	}
	return true, nil
}

func sgExists(d *schema.ResourceData, meta interface{}, id string) (bool, error) {
	sess, err := vpcClient(meta)
	if err != nil {
		return false, err
	}
	getSecurityGroupOptions := &vpcv1.GetSecurityGroupOptions{
		ID: &id,
	}
	_, response, err := sess.GetSecurityGroup(getSecurityGroupOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("Error getting Security Group: %s\n%s", err, response)
	}
	return true, nil
}

func makeIBMISSecurityRuleSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{

		isSecurityGroupRuleDirection: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Direction of traffic to enforce, either inbound or outbound",
		},

		isSecurityGroupRuleIPVersion: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "IP version: ipv4 or ipv6",
		},

		isSecurityGroupRuleRemote: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Security group id: an IP address, a CIDR block, or a single security group identifier",
		},

		isSecurityGroupRuleType: {
			Type:     schema.TypeInt,
			Computed: true,
		},

		isSecurityGroupRuleCode: {
			Type:     schema.TypeInt,
			Computed: true,
		},

		isSecurityGroupRulePortMin: {
			Type:     schema.TypeInt,
			Computed: true,
		},

		isSecurityGroupRulePortMax: {
			Type:     schema.TypeInt,
			Computed: true,
		},

		isSecurityGroupRuleProtocol: {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}
