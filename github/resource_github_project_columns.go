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
		// Update: resourceGithubProjectColumnsUpdate,
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
				ForceNew: true,
			},
			"columns": {
				Type:     schema.TypeList,
				ForceNew: true,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"id": {
							Type:     schema.TypeInt,
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

	projectID := int64(d.Get("project_id").(int))

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
		_, _, err := client.Projects.CreateProjectColumn(context.TODO(), projectID, &opt)
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

	projectID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	opt := &github.ListOptions{PerPage: 10}
	var allColumns []*github.ProjectColumn

	for {
		projectColumns, resp, err := client.Projects.ListProjectColumns(context.TODO(), projectID, opt)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				d.SetId("")
				return nil
			}
			return err
		}

		allColumns = append(allColumns, projectColumns...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	d.Set("project_id", projectID)
	if err := flattenAndSetProjectColumns(d, allColumns); err != nil {
		return fmt.Errorf("Error setting columns: %v", err)
	}

	return nil
}

func flattenAndSetProjectColumns(d *schema.ResourceData, projectColumns []*github.ProjectColumn) error {
	if projectColumns == nil {
		return d.Set("columns", []interface{}{})
	}

	columns := make([]interface{}, 0, len(projectColumns))
	for _, c := range projectColumns {
		v := map[string]interface{}{
			"name": c.GetName(),
			"id":   c.GetID(),
		}
		columns = append(columns, v)
	}

	return d.Set("columns", columns)
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

	ids, err := expandProjectColumnIDs(d)
	if err != nil {
		return err
	}

	for _, id := range ids {
		_, err = client.Projects.DeleteProjectColumn(context.TODO(), id)
		if err != nil {
			return err
		}
	}

	return nil
}

func expandProjectColumnIDs(d *schema.ResourceData) ([]int64, error) {
	v, ok := d.GetOk("columns")
	if !ok {
		return nil, nil
	}

	columnList, ok := v.([]interface{})
	if !ok {
		return nil, nil
	}

	var ids []int64

	for _, cl := range columnList {
		m := cl.(map[string]interface{})

		ids = append(ids, int64(m["id"].(int)))
	}

	return ids, nil
}
