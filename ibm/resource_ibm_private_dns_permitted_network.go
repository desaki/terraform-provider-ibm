package ibm

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	pdnsVpcCRN                     = "vpc_crn"
	pdnsNetworkType                = "type"
	pdnsPermittedNetworkID         = "permitted_network_id"
	pdnsPermittedNetworkCreatedOn  = "created_on"
	pdnsPermittedNetworkModifiedOn = "modified_on"
	pdnsPermittedNetworkState      = "state"
	pdnsPermittedNetwork           = "permitted_network"
)

var allowedNetworkTypes = []string{
	"vpc",
}

func resourceIBMPrivateDNSPermittedNetwork() *schema.Resource {
	return &schema.Resource{
		Create:   resourceIBMPrivateDnsPermittedNetworkCreate,
		Read:     resourceIBMPrivateDnsPermittedNetworkRead,
		Delete:   resourceIBMPrivateDnsPermittedNetworkDelete,
		Exists:   resourceIBMPrivateDnsPermittedNetworkExists,
		Importer: &schema.ResourceImporter{},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			pdnsPermittedNetworkID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Network Id",
			},

			pdnsInstanceID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Instance Id",
			},

			pdnsZoneID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Zone Id",
			},

			pdnsNetworkType: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "vpc",
				ValidateFunc: validateAllowedStringValue([]string{"vpc"}),
				Description:  "Network Type",
			},

			pdnsVpcCRN: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "VPC CRN id",
			},

			pdnsPermittedNetworkCreatedOn: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Network creation date",
			},

			pdnsPermittedNetworkModifiedOn: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Network Modification date",
			},

			pdnsPermittedNetworkState: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Network status",
			},
		},
	}
}

func resourceIBMPrivateDnsPermittedNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(ClientSession).PrivateDnsClientSession()
	if err != nil {
		return err
	}

	instanceID := d.Get(pdnsInstanceID).(string)
	zoneID := d.Get(pdnsZoneID).(string)
	vpcCRN := d.Get(pdnsVpcCRN).(string)
	nwType := d.Get(pdnsNetworkType).(string)
	mk := "private_dns_permitted_network_" + instanceID + zoneID
	ibmMutexKV.Lock(mk)
	defer ibmMutexKV.Unlock(mk)

	createPermittedNetworkOptions := sess.NewCreatePermittedNetworkOptions(instanceID, zoneID)
	permittedNetworkCrn, err := sess.NewPermittedNetworkVpc(vpcCRN)
	if err != nil {
		return err
	}

	createPermittedNetworkOptions.SetPermittedNetwork(permittedNetworkCrn)
	createPermittedNetworkOptions.SetType(nwType)
	response, detail, err := sess.CreatePermittedNetwork(createPermittedNetworkOptions)
	if err != nil {
		return fmt.Errorf("Error creating pdns permitted network:%s\n%s", err, detail)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", instanceID, zoneID, *response.ID))
	d.Set(pdnsPermittedNetworkID, *response.ID)

	return resourceIBMPrivateDnsPermittedNetworkRead(d, meta)
}

func resourceIBMPrivateDnsPermittedNetworkRead(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(ClientSession).PrivateDnsClientSession()
	if err != nil {
		return err
	}

	id_set := strings.Split(d.Id(), "/")
	getPermittedNetworkOptions := sess.NewGetPermittedNetworkOptions(id_set[0], id_set[1], id_set[2])
	response, detail, err := sess.GetPermittedNetwork(getPermittedNetworkOptions)

	if err != nil {
		return fmt.Errorf("Error reading pdns permitted network:%s\n%s", err, detail)
	}

	d.Set("id", response.ID)
	d.Set(pdnsInstanceID, id_set[0])
	d.Set(pdnsZoneID, id_set[1])
	d.Set(pdnsPermittedNetworkID, response.ID)
	d.Set(pdnsPermittedNetworkCreatedOn, response.CreatedOn)
	d.Set(pdnsPermittedNetworkModifiedOn, response.ModifiedOn)
	d.Set(pdnsVpcCRN, response.PermittedNetwork)
	d.Set(pdnsNetworkType, response.Type)
	d.Set(pdnsPermittedNetworkState, response.State)

	return nil
}

func resourceIBMPrivateDnsPermittedNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(ClientSession).PrivateDnsClientSession()
	if err != nil {
		return err
	}

	id_set := strings.Split(d.Id(), "/")
	mk := "private_dns_permitted_network_" + id_set[0] + id_set[1]
	ibmMutexKV.Lock(mk)
	defer ibmMutexKV.Unlock(mk)
	deletePermittedNetworkOptions := sess.NewDeletePermittedNetworkOptions(id_set[0], id_set[1], id_set[2])
	_, response, err := sess.DeletePermittedNetwork(deletePermittedNetworkOptions)

	if err != nil {
		return fmt.Errorf("Error deleting pdns permitted network:%s\n%s", err, response)
	}

	d.SetId("")
	return nil
}

func resourceIBMPrivateDnsPermittedNetworkExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	sess, err := meta.(ClientSession).PrivateDnsClientSession()
	if err != nil {
		return false, err
	}

	id_set := strings.Split(d.Id(), "/")
	mk := "private_dns_permitted_network_" + id_set[0] + id_set[1]
	ibmMutexKV.Lock(mk)
	defer ibmMutexKV.Unlock(mk)
	getPermittedNetworkOptions := sess.NewGetPermittedNetworkOptions(id_set[0], id_set[1], id_set[2])
	_, response, err := sess.GetPermittedNetwork(getPermittedNetworkOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
