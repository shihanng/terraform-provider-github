package github

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubProjectColumn_basic(t *testing.T) {
	randRepoName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	var column github.ProjectColumn

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGithubProjectColumnDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubProjectColumnConfig(randRepoName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubProjectColumnExists("github_project_column.column_1", &column),
					testAccCheckGithubProjectColumnAttributes(&column, &testAccGithubProjectColumnExpectedAttributes{
						Name: "column-1",
					}),
				),
			},
		},
	})
}

func TestAccGithubProjectColumn_importBasic(t *testing.T) {
	randRepoName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGithubProjectColumnDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubProjectColumnConfig(randRepoName),
			},
			{
				ResourceName:      "github_project_column.column_1",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGithubProjectColumnDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_project_column" {
			continue
		}

		columnID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		column, res, err := conn.Projects.GetProjectColumn(context.TODO(), columnID)
		if err == nil {
			if column != nil &&
				column.GetID() == columnID {
				return fmt.Errorf("Project column still exists")
			}
		}
		if res.StatusCode != 404 {
			return err
		}
	}
	return nil
}

func testAccCheckGithubProjectColumnExists(n string, project *github.ProjectColumn) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		columnID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		conn := testAccProvider.Meta().(*Organization).client
		gotColumn, _, err := conn.Projects.GetProjectColumn(context.TODO(), columnID)
		if err != nil {
			return err
		}
		*project = *gotColumn
		return nil
	}
}

type testAccGithubProjectColumnExpectedAttributes struct {
	Name string
}

func testAccCheckGithubProjectColumnAttributes(column *github.ProjectColumn, want *testAccGithubProjectColumnExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *column.Name != want.Name {
			return fmt.Errorf("got project column %q; want %q", *column.Name, want.Name)
		}

		return nil
	}
}

func testAccGithubProjectColumnConfig(repoName string) string {
	return fmt.Sprintf(`
resource "github_repository" "foo" {
  name         = "%[1]s"
  description  = "Terraform acceptance tests"
  homepage_url = "http://example.com/"

  # So that acceptance tests can be run in a github organization
  # with no billing
  private = false

  has_projects  = true
  has_issues    = true
  has_wiki      = true
  has_downloads = true
}

resource "github_repository_project" "test" {
  depends_on = ["github_repository.foo"]

  name       = "test-project"
  repository = "%[1]s"
  body       = "this is a test project"
}

resource "github_project_column" "column_1" {
  project_id = "${github_repository_project.test.id}"
  name       = "column-1"
  position   = "first"
}

`, repoName)
}

func TestValidateGithubProjectColumnPosition(t *testing.T) {
	inputs := []string{
		"first",
		"last",
		"after:abc",
		"after:123",
		"after:",
	}

	for _, in := range inputs {
		_, err := validateGithubProjectColumnPosition(in, "position")
		if err != nil {
			t.Fatal(err)
		}
	}

	if _, err := validateGithubProjectColumnPosition("first:", "position"); err == nil {
		t.Fatal("Expected error, actual: nil")
	}
}

type testLister struct {
	columns []*github.ProjectColumn
}

// ListProjectColumns return the list of columns that has only max of two entries for testing purpose.
func (l *testLister) ListProjectColumns(_ context.Context, _ int64, opts *github.ListOptions) ([]*github.ProjectColumn, *github.Response, error) {
	next := opts.Page + 2

	if next >= len(l.columns) {
		return l.columns[opts.Page:], &github.Response{NextPage: 0}, nil
	}

	return l.columns[opts.Page:next], &github.Response{NextPage: next}, nil
}

func TestGetProjectColumnPosition(t *testing.T) {
	var projectID int64 = 1234

	lister := &testLister{
		columns: []*github.ProjectColumn{
			{ID: github.Int64(0), Name: github.String("first column")},
		},
	}

	_, got, err := getProjectColumn(lister, projectID, 0)
	if err != nil {
		t.Fatal(err)
	}

	if want := "first"; got != want {
		t.Fatalf("got project column position %q; want %q", got, want)
	}

	if _, _, err := getProjectColumn(lister, projectID, 1); err != errProjectColumnNotFound {
		t.Fatalf("Expected error: %v, actual: %v", errProjectColumnNotFound, err)
	}
}

func TestGetProjectColumnPositions(t *testing.T) {
	var projectID int64 = 1234

	lister := &testLister{
		columns: []*github.ProjectColumn{
			{ID: github.Int64(0), Name: github.String("first column")},
			{ID: github.Int64(1), Name: github.String("second column")},
			{ID: github.Int64(2), Name: github.String("third column")},
			{ID: github.Int64(3), Name: github.String("forth column")},
		},
	}

	wantPositions := []string{
		"first",
		"after:0",
		"after:1",
		"last",
	}

	for columnID, want := range wantPositions {
		_, got, err := getProjectColumn(lister, projectID, int64(columnID))
		if err != nil {
			t.Fatal(err)
		}

		if got != want {
			t.Fatalf("got project column position %q; want %q", got, want)
		}
	}
}
