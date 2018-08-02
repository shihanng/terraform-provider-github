package github

import (
	"context"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubOrganizationProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubOrganizationProjectCreate,
		Read:   resourceGithubOrganizationProjectRead,
		Delete: resourceGithubOrganizationProjectDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"body": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceGithubOrganizationProjectCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	o := meta.(*Organization).name
	n := d.Get("name").(string)
	b := d.Get("body").(string)

	options := github.ProjectOptions{
		Name: n,
		Body: b,
	}

	project, _, err := client.Organizations.CreateProject(context.TODO(), o, &options)
	if err != nil {
		return err
	}
	d.SetId(strconv.FormatInt(*project.ID, 10))

	return nil
}

func resourceGithubOrganizationProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	projectID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}

	project, resp, err := client.Projects.GetProject(context.TODO(), projectID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", project.GetName())
	d.Set("body", project.GetBody())

	return nil
}

func resourceGithubOrganizationProjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	projectID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}

	_, err = client.Projects.DeleteProject(context.TODO(), projectID)
	return err
}
