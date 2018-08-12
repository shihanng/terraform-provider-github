package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubProjectColumns() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubProjectColumnsCreate,
		Read:   resourceGithubProjectColumnsRead,
		Update: resourceGithubProjectColumnsUpdate,
		Delete: resourceGithubProjectColumnsDelete,
		// Importer: &schema.ResourceImporter{
		// 	State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
		// 		parts := strings.Split(d.Id(), "/")
		// 		if len(parts) != 2 {
		// 			return nil, fmt.Errorf("Invalid ID specified. Supplied ID must be written as <repository>/<project_id>")
		// 		}
		// 		d.Set("repository", parts[0])
		// 		d.SetId(parts[1])
		// 		return []*schema.ResourceData{d}, nil
		// 	},
		// },

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"columns": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							Default:  false,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceGithubProjectColumnsCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	projectID := d.Get("project_id").(int64)

	opt := &github.ListOptions{PerPage: 10}

	projectColumns, _, err := client.Projects.ListProjectColumns(context.TODO(), projectID, opt)
	if err != nil {
		return err
	}

	if len(projectColumns) > 0 {
		return errors.New("Refuse to create new columns as project alreadys contains columns. Use import if you want to update.")
	}

	opts, err := expandProjectColumns(d)
	if err != nil {
		return err
	}

	for _, opt := range opts {
		column, _, err := client.Projects.CreateProjectColumn(context.TODO(), projectID, opt)
		if err != nil {
			return err
		}

	}

	d.SetId(strconv.FormatInt(projectID, 10))

	return resourceGithubProjectColumnsRead(d, meta)
}

func expandProjectColumns(d *schema.ResourceData) ([]github.ProjectColumnOptions, error) {
	v, ok := d.GetOk("columns")
	if !ok {
		return nil, nil
	}

	columnList, ok := v.([]interface{})
	if !ok {
		return nil, nil
	}

	var opts []github.ProjectColumnOptions

	for _, cl := range columnList {
		m := cl.(map[string]interface{})

		opts = append(opts, github.ProjectColumnOptions{
			Name: m["name"].(string),
		})
	}

	return opts, nil
}

func resourceGithubProjectColumnsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	orgName := meta.(*Organization).name

	projectID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
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
	d.Set("url", fmt.Sprintf("https://github.com/%s/%s/projects/%d",
		orgName, d.Get("repository"), project.GetNumber()))

	return nil
}

func resourceGithubProjectColumnsUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	options := github.ProjectOptions{
		Name: d.Get("name").(string),
		Body: d.Get("body").(string),
	}

	projectID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	_, _, err = client.Projects.UpdateProject(context.TODO(), projectID, &options)
	if err != nil {
		return err
	}

	return resourceGithubProjectColumnsRead(d, meta)
}

func resourceGithubProjectColumnsDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	projectID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	_, err = client.Projects.DeleteProject(context.TODO(), projectID)
	return err
}
