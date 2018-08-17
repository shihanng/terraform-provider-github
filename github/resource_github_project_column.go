package github

import (
	"context"
	"errors"
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
				client := meta.(*Organization).client

				columnID, err := strconv.ParseInt(d.Id(), 10, 64)
				if err != nil {
					return nil, unconvertibleIdErr(d.Id(), err)
				}

				column, _, err := client.Projects.GetProjectColumn(context.TODO(), columnID)
				if err != nil {
					return nil, err
				}

				projectURL := column.GetProjectURL()
				projectID := strings.TrimPrefix(projectURL, "https://api.github.com/projects/")

				d.Set("project_id", projectID)
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
			"position": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateGithubProjectColumnPosition,
			},
		},
	}
}

func validateGithubProjectColumnPosition(v interface{}, k string) (ws []string, errors []error) {
	position := v.(string)

	if position == "first" || position == "last" || strings.HasPrefix(position, "after:") {
		return
	}

	errors = append(errors, fmt.Errorf("Github: %s can only be one of 'first', 'last', or 'after:<column_id>'", k))
	return
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

	columnID := *column.ID
	d.SetId(strconv.FormatInt(columnID, 10))

	if position := d.Get("position").(string); position != "" {
		options := github.ProjectColumnMoveOptions{
			Position: position,
		}

		_, err = client.Projects.MoveProjectColumn(context.TODO(), columnID, &options)
		if err != nil {
			return err
		}
	}

	return resourceGithubProjectColumnRead(d, meta)
}

func resourceGithubProjectColumnRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	columnID, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}

	projectIDStr := d.Get("project_id").(string)
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return unconvertibleIdErr(projectIDStr, err)
	}

	column, position, err := getProjectColumn(client.Projects, projectID, columnID)
	if err != nil {
		if err == errProjectColumnNotFound {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", column.GetName())
	d.Set("position", position)

	return nil
}

var errProjectColumnNotFound = errors.New("not found")

type projectColumnsLister interface {
	ListProjectColumns(context.Context, int64, *github.ListOptions) ([]*github.ProjectColumn, *github.Response, error)
}

func getProjectColumn(lister projectColumnsLister, projectID, columnID int64) (*github.ProjectColumn, string, error) {
	listOptions := &github.ListOptions{PerPage: 10}

	var projectColumns []*github.ProjectColumn

	for {
		columns, resp, err := lister.ListProjectColumns(context.TODO(), projectID, listOptions)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				return nil, "", errProjectColumnNotFound
			}
			return nil, "", err
		}

		for i, c := range columns {
			if c.GetID() != columnID {
				continue
			}

			// Computing the position of the found column.
			l := len(projectColumns)
			if l == 0 {
				return c, "first", nil
			}

			if resp.NextPage != 0 {
				return c, fmt.Sprintf("after:%d", projectColumns[l-1].GetID()), nil
			}

			if len(columns)-1 == i {
				return c, "last", nil
			}

			return c, fmt.Sprintf("after:%d", projectColumns[l-1].GetID()), nil
		}

		projectColumns = append(projectColumns, columns...)
		if resp.NextPage == 0 {
			break
		}

		listOptions.Page = resp.NextPage
	}

	return nil, "", errProjectColumnNotFound
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

	if position := d.Get("position").(string); position != "" {
		options := github.ProjectColumnMoveOptions{
			Position: position,
		}

		_, err = client.Projects.MoveProjectColumn(context.TODO(), columnID, &options)
		if err != nil {
			return err
		}
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
