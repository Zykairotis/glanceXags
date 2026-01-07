package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	apiBaseURL          = "https://api.todoist.com/rest/v2"
	rateLimitPerQuarter = 1000
	rateLimitDuration   = 15 * time.Minute
)

// Client represents a Todoist API client
type Client struct {
	apiToken     string
	httpClient   *http.Client
	requestCount int
	quarterStart time.Time
}

// NewClient creates a new Todoist API client
func NewClient(apiToken string) *Client {
	return &Client{
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		quarterStart: time.Now(),
	}
}

// doRequest performs an HTTP request with proper headers and error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, v any) error {
	// Check rate limit
	c.checkRateLimit()

	url := apiBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	c.requestCount++

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if v != nil {
			if err := json.Unmarshal(respBody, v); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}
		}
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: invalid API token")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: insufficient permissions")
	case http.StatusNotFound:
		return fmt.Errorf("not found: resource does not exist")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded: %s", string(respBody))
	default:
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
}

// checkRateLimit implements simple rate limiting awareness
func (c *Client) checkRateLimit() {
	now := time.Now()
	elapsed := now.Sub(c.quarterStart)

	if elapsed >= rateLimitDuration {
		c.quarterStart = now
		c.requestCount = 0
	}

	if c.requestCount >= rateLimitPerQuarter {
		slog.Warn("Todoist rate limit approaching", "requests", c.requestCount, "period", rateLimitDuration)
	}
}

//
// Data Structures
//

// Task represents a Todoist task
type Task struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	IsCompleted bool      `json:"is_completed"`
	Order       int       `json:"order"`
	Priority    int       `json:"priority"`
	ProjectID   string    `json:"project_id"`
	SectionID   string    `json:"section_id"`
	ParentID    string    `json:"parent_id"`
	CreatorID   string    `json:"creator_id"`
	CreatedAt   time.Time `json:"created_at"`
	Due         *DueInfo  `json:"due"`
	Deadline    *Deadline `json:"deadline"`
	Duration    *Duration `json:"duration"`
	Labels      []string  `json:"labels"`
	CommentCount int      `json:"comment_count"`
	AssigneeID  string    `json:"assignee_id"`
	URL         string    `json:"url"`
}

// DueInfo represents task due information
type DueInfo struct {
	Date        string `json:"date"`
	IsRecurring bool   `json:"is_recurring"`
	DateTime    string `json:"datetime"`
	String      string `json:"string"`
	Timezone    string `json:"timezone"`
	Lang        string `json:"lang"`
}

// Deadline represents task deadline
type Deadline struct {
	Date string `json:"date"`
}

// Duration represents task duration
type Duration struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

// Project represents a Todoist project
type Project struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Color          string `json:"color"`
	ParentID       string `json:"parent_id"`
	Order          int    `json:"order"`
	CommentCount   int    `json:"comment_count"`
	IsShared       bool   `json:"is_shared"`
	IsFavorite     bool   `json:"is_favorite"`
	IsInboxProject bool   `json:"is_inbox_project"`
	IsTeamInbox    bool   `json:"is_team_inbox"`
	ViewStyle      string `json:"view_style"`
	URL            string `json:"url"`
}

// Section represents a Todoist section
type Section struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Order     int    `json:"order"`
	Name      string `json:"name"`
}

// Label represents a Todoist label
type Label struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	Order      int    `json:"order"`
	IsFavorite bool   `json:"is_favorite"`
}

// Comment represents a comment on a task or project
type Comment struct {
	ID        string       `json:"id"`
	Content   string       `json:"content"`
	PostedAt  time.Time    `json:"posted_at"`
	Attachment *Attachment `json:"attachment"`
	TaskID    string       `json:"task_id"`
	ProjectID string       `json:"project_id"`
}

// Attachment represents a file attachment
type Attachment struct {
	ResourceType string `json:"resource_type"`
	FileType     string `json:"file_type"`
	FileName     string `json:"file_name"`
	FileURL      string `json:"file_url"`
}

// Collaborator represents a project collaborator
type Collaborator struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

//
// Task API Methods
//

// GetTasksOptions defines options for filtering tasks
type GetTasksOptions struct {
	ProjectID string
	SectionID string
	Label     string
	Filter    string
	Lang      string
	IDs       []string
}

// GetTasks retrieves all tasks with optional filters
func (c *Client) GetTasks(ctx context.Context, opts *GetTasksOptions) ([]Task, error) {
	var tasks []Task

	path := "/tasks"
	if opts != nil {
		params := make([]string, 0)
		if opts.ProjectID != "" {
			params = append(params, "project_id="+opts.ProjectID)
		}
		if opts.SectionID != "" {
			params = append(params, "section_id="+opts.SectionID)
		}
		if opts.Label != "" {
			params = append(params, "label="+opts.Label)
		}
		if opts.Filter != "" {
			params = append(params, "filter="+opts.Filter)
		}
		if opts.Lang != "" {
			params = append(params, "lang="+opts.Lang)
		}
		if len(opts.IDs) > 0 {
			params = append(params, "ids="+strings.Join(opts.IDs, ","))
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}
	}

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &tasks); err != nil {
		return nil, fmt.Errorf("getting tasks: %w", err)
	}

	return tasks, nil
}

// GetTask retrieves a single task by ID
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	var task Task
	if err := c.doRequest(ctx, http.MethodGet, "/tasks/"+taskID, nil, &task); err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	return &task, nil
}

// CreateTaskRequest defines options for creating a task
type CreateTaskRequest struct {
	Content     string   `json:"content"`
	Description string   `json:"description,omitempty"`
	ProjectID   string   `json:"project_id,omitempty"`
	SectionID   string   `json:"section_id,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"`
	Order       int      `json:"order,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	DueString   string   `json:"due_string,omitempty"`
	DueDate     string   `json:"due_date,omitempty"`
	DueDatetime string   `json:"due_datetime,omitempty"`
	DueLang     string   `json:"due_lang,omitempty"`
	AssigneeID  string   `json:"assignee_id,omitempty"`
	Duration    int      `json:"duration,omitempty"`
	DurationUnit string  `json:"duration_unit,omitempty"`
	DeadlineDate string  `json:"deadline_date,omitempty"`
}

// CreateTask creates a new task
func (c *Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var task Task
	if err := c.doRequest(ctx, http.MethodPost, "/tasks", bytes.NewReader(body), &task); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	return &task, nil
}

// UpdateTaskRequest defines options for updating a task
type UpdateTaskRequest struct {
	Content     *string  `json:"content,omitempty"`
	Description *string  `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	DueString   *string  `json:"due_string,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
	DueDatetime *string  `json:"due_datetime,omitempty"`
	DueLang     *string  `json:"due_lang,omitempty"`
	AssigneeID  *string  `json:"assignee_id,omitempty"`
	Duration    *int     `json:"duration,omitempty"`
	DurationUnit *string `json:"duration_unit,omitempty"`
	DeadlineDate *string `json:"deadline_date,omitempty"`
}

// UpdateTask updates an existing task
func (c *Client) UpdateTask(ctx context.Context, taskID string, req *UpdateTaskRequest) (*Task, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var task Task
	if err := c.doRequest(ctx, http.MethodPost, "/tasks/"+taskID, bytes.NewReader(body), &task); err != nil {
		return nil, fmt.Errorf("updating task: %w", err)
	}

	return &task, nil
}

// CloseTask marks a task as complete
func (c *Client) CloseTask(ctx context.Context, taskID string) error {
	return c.doRequest(ctx, http.MethodPost, "/tasks/"+taskID+"/close", nil, nil)
}

// ReopenTask reopens a completed task
func (c *Client) ReopenTask(ctx context.Context, taskID string) error {
	return c.doRequest(ctx, http.MethodPost, "/tasks/"+taskID+"/reopen", nil, nil)
}

// DeleteTask deletes a task
func (c *Client) DeleteTask(ctx context.Context, taskID string) error {
	return c.doRequest(ctx, http.MethodDelete, "/tasks/"+taskID, nil, nil)
}

//
// Project API Methods
//

// GetProjects retrieves all projects
func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	if err := c.doRequest(ctx, http.MethodGet, "/projects", nil, &projects); err != nil {
		return nil, fmt.Errorf("getting projects: %w", err)
	}
	return projects, nil
}

// GetProject retrieves a single project by ID
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	var project Project
	if err := c.doRequest(ctx, http.MethodGet, "/projects/"+projectID, nil, &project); err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return &project, nil
}

// CreateProjectRequest defines options for creating a project
type CreateProjectRequest struct {
	Name       string `json:"name"`
	ParentID   string `json:"parent_id,omitempty"`
	Color      string `json:"color,omitempty"`
	IsFavorite bool   `json:"is_favorite,omitempty"`
	ViewStyle  string `json:"view_style,omitempty"`
}

// CreateProject creates a new project
func (c *Client) CreateProject(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var project Project
	if err := c.doRequest(ctx, http.MethodPost, "/projects", bytes.NewReader(body), &project); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	return &project, nil
}

// UpdateProjectRequest defines options for updating a project
type UpdateProjectRequest struct {
	Name       *string `json:"name,omitempty"`
	Color      *string `json:"color,omitempty"`
	IsFavorite *bool   `json:"is_favorite,omitempty"`
	ViewStyle  *string `json:"view_style,omitempty"`
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(ctx context.Context, projectID string, req *UpdateProjectRequest) (*Project, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var project Project
	if err := c.doRequest(ctx, http.MethodPost, "/projects/"+projectID, bytes.NewReader(body), &project); err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	return &project, nil
}

// ArchiveProject archives a project
func (c *Client) ArchiveProject(ctx context.Context, projectID string) error {
	return c.doRequest(ctx, http.MethodPost, "/projects/"+projectID+"/archive", nil, nil)
}

// UnarchiveProject unarchives a project
func (c *Client) UnarchiveProject(ctx context.Context, projectID string) error {
	return c.doRequest(ctx, http.MethodPost, "/projects/"+projectID+"/unarchive", nil, nil)
}

// DeleteProject deletes a project
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	return c.doRequest(ctx, http.MethodDelete, "/projects/"+projectID, nil, nil)
}

// GetProjectCollaborators retrieves collaborators for a project
func (c *Client) GetProjectCollaborators(ctx context.Context, projectID string) ([]Collaborator, error) {
	var collaborators []Collaborator
	if err := c.doRequest(ctx, http.MethodGet, "/projects/"+projectID+"/collaborators", nil, &collaborators); err != nil {
		return nil, fmt.Errorf("getting collaborators: %w", err)
	}
	return collaborators, nil
}

//
// Section API Methods
//

// GetSections retrieves all sections, optionally filtered by project
func (c *Client) GetSections(ctx context.Context, projectID string) ([]Section, error) {
	path := "/sections"
	if projectID != "" {
		path += "?project_id=" + projectID
	}

	var sections []Section
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &sections); err != nil {
		return nil, fmt.Errorf("getting sections: %w", err)
	}
	return sections, nil
}

// GetSection retrieves a single section by ID
func (c *Client) GetSection(ctx context.Context, sectionID string) (*Section, error) {
	var section Section
	if err := c.doRequest(ctx, http.MethodGet, "/sections/"+sectionID, nil, &section); err != nil {
		return nil, fmt.Errorf("getting section: %w", err)
	}
	return &section, nil
}

// CreateSectionRequest defines options for creating a section
type CreateSectionRequest struct {
	Name     string `json:"name"`
	ProjectID string `json:"project_id,omitempty"`
	Order    int    `json:"order,omitempty"`
}

// CreateSection creates a new section
func (c *Client) CreateSection(ctx context.Context, req *CreateSectionRequest) (*Section, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var section Section
	if err := c.doRequest(ctx, http.MethodPost, "/sections", bytes.NewReader(body), &section); err != nil {
		return nil, fmt.Errorf("creating section: %w", err)
	}

	return &section, nil
}

// UpdateSectionRequest defines options for updating a section
type UpdateSectionRequest struct {
	Name *string `json:"name,omitempty"`
}

// UpdateSection updates an existing section
func (c *Client) UpdateSection(ctx context.Context, sectionID string, req *UpdateSectionRequest) (*Section, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var section Section
	if err := c.doRequest(ctx, http.MethodPost, "/sections/"+sectionID, bytes.NewReader(body), &section); err != nil {
		return nil, fmt.Errorf("updating section: %w", err)
	}

	return &section, nil
}

// DeleteSection deletes a section
func (c *Client) DeleteSection(ctx context.Context, sectionID string) error {
	return c.doRequest(ctx, http.MethodDelete, "/sections/"+sectionID, nil, nil)
}

//
// Comment API Methods
//

// GetCommentsOptions defines options for getting comments
type GetCommentsOptions struct {
	TaskID    string
	ProjectID string
}

// GetComments retrieves comments for a task or project
func (c *Client) GetComments(ctx context.Context, opts *GetCommentsOptions) ([]Comment, error) {
	path := "/comments"
	if opts != nil {
		if opts.TaskID != "" {
			path += "?task_id=" + opts.TaskID
		} else if opts.ProjectID != "" {
			path += "?project_id=" + opts.ProjectID
		}
	}

	var comments []Comment
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &comments); err != nil {
		return nil, fmt.Errorf("getting comments: %w", err)
	}
	return comments, nil
}

// GetComment retrieves a single comment by ID
func (c *Client) GetComment(ctx context.Context, commentID string) (*Comment, error) {
	var comment Comment
	if err := c.doRequest(ctx, http.MethodGet, "/comments/"+commentID, nil, &comment); err != nil {
		return nil, fmt.Errorf("getting comment: %w", err)
	}
	return &comment, nil
}

// CreateCommentRequest defines options for creating a comment
type CreateCommentRequest struct {
	Content   string      `json:"content"`
	TaskID    string      `json:"task_id,omitempty"`
	ProjectID string      `json:"project_id,omitempty"`
	Attachment *Attachment `json:"attachment,omitempty"`
}

// CreateComment creates a new comment
func (c *Client) CreateComment(ctx context.Context, req *CreateCommentRequest) (*Comment, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if req.TaskID == "" && req.ProjectID == "" {
		return nil, fmt.Errorf("either task_id or project_id is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var comment Comment
	if err := c.doRequest(ctx, http.MethodPost, "/comments", bytes.NewReader(body), &comment); err != nil {
		return nil, fmt.Errorf("creating comment: %w", err)
	}

	return &comment, nil
}

// UpdateCommentContent updates a comment's content
func (c *Client) UpdateCommentContent(ctx context.Context, commentID, content string) (*Comment, error) {
	req := map[string]string{
		"content": content,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var comment Comment
	if err := c.doRequest(ctx, http.MethodPost, "/comments/"+commentID, bytes.NewReader(body), &comment); err != nil {
		return nil, fmt.Errorf("updating comment: %w", err)
	}

	return &comment, nil
}

// DeleteComment deletes a comment
func (c *Client) DeleteComment(ctx context.Context, commentID string) error {
	return c.doRequest(ctx, http.MethodDelete, "/comments/"+commentID, nil, nil)
}

//
// Label API Methods
//

// GetLabels retrieves all personal labels
func (c *Client) GetLabels(ctx context.Context) ([]Label, error) {
	var labels []Label
	if err := c.doRequest(ctx, http.MethodGet, "/labels", nil, &labels); err != nil {
		return nil, fmt.Errorf("getting labels: %w", err)
	}
	return labels, nil
}

// GetLabel retrieves a single label by ID
func (c *Client) GetLabel(ctx context.Context, labelID string) (*Label, error) {
	var label Label
	if err := c.doRequest(ctx, http.MethodGet, "/labels/"+labelID, nil, &label); err != nil {
		return nil, fmt.Errorf("getting label: %w", err)
	}
	return &label, nil
}

// CreateLabelRequest defines options for creating a label
type CreateLabelRequest struct {
	Name      string `json:"name"`
	Order     int    `json:"order,omitempty"`
	Color     string `json:"color,omitempty"`
	IsFavorite bool  `json:"is_favorite,omitempty"`
}

// CreateLabel creates a new label
func (c *Client) CreateLabel(ctx context.Context, req *CreateLabelRequest) (*Label, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var label Label
	if err := c.doRequest(ctx, http.MethodPost, "/labels", bytes.NewReader(body), &label); err != nil {
		return nil, fmt.Errorf("creating label: %w", err)
	}

	return &label, nil
}

// UpdateLabelRequest defines options for updating a label
type UpdateLabelRequest struct {
	Name      *string `json:"name,omitempty"`
	Order     *int    `json:"order,omitempty"`
	Color     *string `json:"color,omitempty"`
	IsFavorite *bool  `json:"is_favorite,omitempty"`
}

// UpdateLabel updates an existing label
func (c *Client) UpdateLabel(ctx context.Context, labelID string, req *UpdateLabelRequest) (*Label, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var label Label
	if err := c.doRequest(ctx, http.MethodPost, "/labels/"+labelID, bytes.NewReader(body), &label); err != nil {
		return nil, fmt.Errorf("updating label: %w", err)
	}

	return &label, nil
}

// DeleteLabel deletes a label
func (c *Client) DeleteLabel(ctx context.Context, labelID string) error {
	return c.doRequest(ctx, http.MethodDelete, "/labels/"+labelID, nil, nil)
}

// GetSharedLabels retrieves all shared labels
func (c *Client) GetSharedLabels(ctx context.Context, omitPersonal bool) ([]string, error) {
	path := "/labels/shared"
	if omitPersonal {
		path += "?omit_personal=true"
	}

	var labels []string
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &labels); err != nil {
		return nil, fmt.Errorf("getting shared labels: %w", err)
	}
	return labels, nil
}

// RenameSharedLabelRequest defines options for renaming a shared label
type RenameSharedLabelRequest struct {
	Name   string `json:"name"`
	NewName string `json:"new_name"`
}

// RenameSharedLabel renames all instances of a shared label
func (c *Client) RenameSharedLabel(ctx context.Context, name, newName string) error {
	req := RenameSharedLabelRequest{
		Name:   name,
		NewName: newName,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	return c.doRequest(ctx, http.MethodPost, "/labels/shared/rename", bytes.NewReader(body), nil)
}

// RemoveSharedLabelRequest defines options for removing a shared label
type RemoveSharedLabelRequest struct {
	Name string `json:"name"`
}

// RemoveSharedLabel removes a shared label from all tasks
func (c *Client) RemoveSharedLabel(ctx context.Context, name string) error {
	req := RemoveSharedLabelRequest{
		Name: name,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	return c.doRequest(ctx, http.MethodPost, "/labels/shared/remove", bytes.NewReader(body), nil)
}

// GetRequestCount returns the current request count for rate limit tracking
func (c *Client) GetRequestCount() int {
	return c.requestCount
}

// Color constants for Todoist
const (
	ColorBerry   = "#b8256f"
	ColorRed     = "#db4035"
	ColorOrange  = "#ff9933"
	ColorYellow  = "#fad000"
	ColorOlive   = "#afb83b"
	ColorGreen   = "#7ecc49"
	ColorCyan    = "#3a8eec"
	ColorBlue    = "#0656bf"
	ColorPurple  = "#654982"
	ColorPink    = "#aa33d1"
	ColorGray    = "#898989"
	ColorCharcoal = "#545454"
)
