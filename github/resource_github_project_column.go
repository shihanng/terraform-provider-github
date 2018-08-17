package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubProjectColumn() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubProjectColumnCreate,
		Read:   resourceGithubProjectColumnRead,
		Update: resourceGithubProjectColumnUpdate,
		Delete: resourceGithubProjectColumnDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 2 {
					return nil, fmt.Errorf("Invalid ID specified. Supplied ID must be written as <repository>/<project_id>")
				}
				d.Set("repository", parts[0])
				d.SetId(parts[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceGithubProjectColumnCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	options := github.ProjectColumnOptions{
		Name: d.Get("name").(string),
	}

	projectIDStr := d.Get("project_id").(string)
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return unconvertibleIdErr(projectIDStr, err)
	}

	column, _, err := client.Projects.CreateProjectColumn(context.TODO(),
		projectID,
		&options,
	)
	if err != nil {
		return err
	}
	d.SetId(strconv.FormatInt(*column.ID, 10))

	return resourceGithubProjectColumnRead(d, meta)
}

func resourceGithubProjectColumnRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	columnID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	column, resp, err := client.Projects.GetProjectColumn(context.TODO(), columnID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", column.GetName())

	return nil
}

func resourceGithubProjectColumnUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	options := github.ProjectColumnOptions{
		Name: d.Get("name").(string),
	}

	columnID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	_, _, err = client.Projects.UpdateProjectColumn(context.TODO(), columnID, &options)
	if err != nil {
		return err
	}

	return resourceGithubProjectColumnRead(d, meta)
}

func resourceGithubProjectColumnDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	columnID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	_, err = client.Projects.DeleteProjectColumn(context.TODO(), columnID)
	return err
}
