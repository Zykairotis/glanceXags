package glance

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/glanceapp/glance/internal/glance/todoist"
)

var todoistWidgetTemplate = mustParseTemplate("todoist.html", "widget-base.html")

// todoistWidget implements the Todoist integration widget
type todoistWidget struct {
	widgetBase `yaml:",inline"`

	// Authentication
	APIToken string `yaml:"api-token"`

	// Task Filtering
	Filter      string `yaml:"filter"`
	ProjectID   string `yaml:"project-id"`
	SectionID   string `yaml:"section-id"`
	Label       string `yaml:"label"`
	PriorityMin int    `yaml:"priority-min"`
	DueFilter   string `yaml:"due-filter"`
	Lang        string `yaml:"lang"`
	IDs         string `yaml:"ids"`
	Assignee    string `yaml:"assignee"`

	// Display Options
	ShowCompleted   bool `yaml:"show-completed"`
	ShowSubtasks    bool `yaml:"show-subtasks"`
	Limit           int  `yaml:"limit"`
	CollapseAfter   int  `yaml:"collapse-after"`
	CompactMode     bool `yaml:"compact-mode"`
	HideDescription bool `yaml:"hide-description"`

	// Task Creation Defaults
	DefaultProjectID string `yaml:"default-project-id"`
	DefaultSectionID string `yaml:"default-section-id"`
	DefaultPriority  int    `yaml:"default-priority"`
	DefaultLabels    string `yaml:"default-labels"`

	// API client (runtime only)
	client *todoist.Client `yaml:"-"`

	// Data
	Tasks        []todoistTaskView `yaml:"-"`
	Projects     []todoist.Project `yaml:"-"`
	Labels       []todoist.Label   `yaml:"-"`
	ErrorMessage string            `yaml:"-"`
}

// todoistTaskView represents a task with its project and labels for display
type todoistTaskView struct {
	Task        *todoist.Task
	Project     *todoist.Project
	Labels      []todoist.Label
	HasSubtasks bool
	Subtasks    []todoistTaskView // Nested subtasks for rendering
}

func (widget *todoistWidget) initialize() error {
	widget.withTitle("Todoist").withCacheDuration(5 * time.Minute)

	if widget.APIToken == "" {
		return fmt.Errorf("api-token is required for Todoist widget")
	}

	widget.client = todoist.NewClient(widget.APIToken)

	if widget.Limit <= 0 {
		widget.Limit = 20
	}

	if widget.CollapseAfter == 0 || widget.CollapseAfter < -1 {
		widget.CollapseAfter = 5
	}

	if widget.DefaultPriority < 1 || widget.DefaultPriority > 4 {
		widget.DefaultPriority = 1
	}

	return nil
}

func (widget *todoistWidget) update(ctx context.Context) {
	if widget.client == nil {
		widget.Error = fmt.Errorf("Todoist client not initialized")
		widget.ContentAvailable = false
		return
	}

	// Build get tasks options
	opts := &todoist.GetTasksOptions{
		ProjectID: widget.ProjectID,
		SectionID: widget.SectionID,
		Label:     widget.Label,
		Filter:    widget.Filter,
		Lang:      widget.Lang,
	}

	// Parse IDs if provided
	if widget.IDs != "" {
		ids := strings.Split(widget.IDs, ",")
		for i := range ids {
			ids[i] = strings.TrimSpace(ids[i])
		}
		opts.IDs = ids
	}

	tasks, err := widget.client.GetTasks(ctx, opts)
	if err != nil {
		slog.Error("Failed to fetch Todoist tasks", "error", err)
		widget.Error = fmt.Errorf("failed to fetch tasks: %w", err)
		widget.ContentAvailable = false
		return
	}

	// Filter out completed tasks if configured
	if !widget.ShowCompleted {
		filtered := make([]todoist.Task, 0, len(tasks))
		for i := range tasks {
			if !tasks[i].IsCompleted {
				filtered = append(filtered, tasks[i])
			}
		}
		tasks = filtered
	}

	// Apply priority filter
	if widget.PriorityMin > 0 && widget.PriorityMin <= 4 {
		filtered := make([]todoist.Task, 0, len(tasks))
		for i := range tasks {
			if tasks[i].Priority >= widget.PriorityMin {
				filtered = append(filtered, tasks[i])
			}
		}
		tasks = filtered
	}

	// Apply due date filter
	if widget.DueFilter != "" {
		now := time.Now()
		today := now.Format("2006-01-02")
		filtered := make([]todoist.Task, 0, len(tasks))
		for i := range tasks {
			task := tasks[i]
			shouldInclude := false

			switch widget.DueFilter {
			case "today":
				if task.Due != nil && task.Due.Date == today {
					shouldInclude = true
				}
			case "overdue":
				if task.Due != nil && task.Due.Date < today {
					shouldInclude = true
				}
			case "upcoming":
				if task.Due != nil && task.Due.Date >= today {
					shouldInclude = true
				}
			case "none":
				if task.Due == nil || task.Due.Date == "" {
					shouldInclude = true
				}
			case "all":
				shouldInclude = true
			default:
				shouldInclude = true
			}

			if shouldInclude {
				filtered = append(filtered, task)
			}
		}
		tasks = filtered
	}

	// Sort tasks by due date and priority
	sort.Slice(tasks, func(i, j int) bool {
		// First sort by completion status
		if tasks[i].IsCompleted != tasks[j].IsCompleted {
			return !tasks[i].IsCompleted
		}

		// Then by priority (higher priority first)
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority > tasks[j].Priority
		}

		// Then by due date
		var iDue, jDue string
		if tasks[i].Due != nil {
			iDue = tasks[i].Due.Date
		}
		if tasks[j].Due != nil {
			jDue = tasks[j].Due.Date
		}

		if iDue != jDue {
			if iDue == "" {
				return false
			}
			if jDue == "" {
				return true
			}
			return iDue < jDue
		}

		// Finally by order
		return tasks[i].Order < tasks[j].Order
	})

	// Apply limit
	if len(tasks) > widget.Limit {
		tasks = tasks[:widget.Limit]
	}

	// Fetch projects and labels for context
	projectsMap := make(map[string]*todoist.Project)
	labelsMap := make(map[string]*todoist.Label)

	if widget.ProjectID != "" || widget.Filter == "" {
		projects, err := widget.client.GetProjects(ctx)
		if err == nil {
			for i := range projects {
				p := &projects[i]
				projectsMap[p.ID] = p
			}
			widget.Projects = projects
		}
	}

	labels, err := widget.client.GetLabels(ctx)
	if err == nil {
		for i := range labels {
			l := &labels[i]
			labelsMap[l.Name] = l
		}
		widget.Labels = labels
	}

	// Build task views with project and label context
	taskViews := make([]todoistTaskView, 0, len(tasks))
	parentIDs := make(map[string]bool)

	for i := range tasks {
		task := &tasks[i]
		if task.ParentID != "" {
			parentIDs[task.ParentID] = true
		}

		var project *todoist.Project
		if task.ProjectID != "" {
			project = projectsMap[task.ProjectID]
		}

		taskLabels := make([]todoist.Label, 0, len(task.Labels))
		for _, labelName := range task.Labels {
			if label, ok := labelsMap[labelName]; ok {
				taskLabels = append(taskLabels, *label)
			}
		}

		taskViews = append(taskViews, todoistTaskView{
			Task:    task,
			Project: project,
			Labels:  taskLabels,
		})
	}

	// Mark tasks with subtasks
	for i := range taskViews {
		if _, hasSubtasks := parentIDs[taskViews[i].Task.ID]; hasSubtasks {
			taskViews[i].HasSubtasks = true
		}
	}

	// Group subtasks under their parents
	taskViewsMap := make(map[string]*todoistTaskView)
	for i := range taskViews {
		taskViewsMap[taskViews[i].Task.ID] = &taskViews[i]
	}

	// Build nested structure - only keep top-level tasks, attach subtasks to their parents
	topLevelTasks := make([]todoistTaskView, 0)
	for i := range taskViews {
		task := &taskViews[i]
		if task.Task.ParentID == "" {
			// This is a top-level task
			topLevelTasks = append(topLevelTasks, *task)
		} else {
			// This is a subtask - attach to parent
			if parent, ok := taskViewsMap[task.Task.ParentID]; ok {
				parent.Subtasks = append(parent.Subtasks, *task)
			}
		}
	}

	// Update the parent references in topLevelTasks with their populated subtasks
	for i := range topLevelTasks {
		if original, ok := taskViewsMap[topLevelTasks[i].Task.ID]; ok {
			topLevelTasks[i].Subtasks = original.Subtasks
		}
	}

	widget.Tasks = topLevelTasks
	widget.ContentAvailable = true
	widget.Error = nil
}

func (widget *todoistWidget) Render() template.HTML {
	return widget.renderTemplate(widget, todoistWidgetTemplate)
}

// handleRequest implements HTTP handlers for Todoist operations
func (widget *todoistWidget) handleRequest(w http.ResponseWriter, r *http.Request) {
	if widget.client == nil {
		http.Error(w, "widget not initialized", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	path := strings.TrimPrefix(r.URL.Path, "/api/widgets/"+fmt.Sprint(widget.ID))

	switch r.Method {
	case http.MethodGet:
		// GET comments for a task: /tasks/{id}/comments
		if strings.Contains(path, "/comments") {
			parts := strings.Split(strings.TrimPrefix(path, "/tasks/"), "/")
			if len(parts) < 2 {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			taskID := parts[0]

			comments, err := widget.client.GetComments(ctx, &todoist.GetCommentsOptions{TaskID: taskID})
			if err != nil {
				slog.Error("Failed to get Todoist comments", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(comments)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)

	case http.MethodPost:
		switch {
		// Close task: /tasks/{id}/close
		case strings.HasSuffix(path, "/close"):
			parts := strings.Split(path, "/")
			if len(parts) < 3 {
				http.Error(w, "invalid task ID", http.StatusBadRequest)
				return
			}
			taskID := parts[len(parts)-2]
			if err := widget.client.CloseTask(ctx, taskID); err != nil {
				slog.Error("Failed to close Todoist task", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			widget.scheduleEarlyUpdate()
			w.WriteHeader(http.StatusNoContent)

		// Reopen task: /tasks/{id}/reopen
		case strings.HasSuffix(path, "/reopen"):
			parts := strings.Split(path, "/")
			if len(parts) < 3 {
				http.Error(w, "invalid task ID", http.StatusBadRequest)
				return
			}
			taskID := parts[len(parts)-2]
			if err := widget.client.ReopenTask(ctx, taskID); err != nil {
				slog.Error("Failed to reopen Todoist task", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			widget.scheduleEarlyUpdate()
			w.WriteHeader(http.StatusNoContent)

		// Create comment: /tasks/{id}/comments
		case strings.HasSuffix(path, "/comments"):
			parts := strings.Split(strings.TrimPrefix(path, "/tasks/"), "/")
			if len(parts) < 2 {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			taskID := parts[0]

			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			comment, err := widget.client.CreateComment(ctx, &todoist.CreateCommentRequest{
				TaskID:  taskID,
				Content: req.Content,
			})
			if err != nil {
				slog.Error("Failed to create Todoist comment", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(comment)

		// Create task: /tasks
		case path == "/tasks" || path == "/tasks/":
			var req todoist.CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Apply defaults if not specified
			if req.ProjectID == "" && widget.DefaultProjectID != "" {
				req.ProjectID = widget.DefaultProjectID
			}
			if req.Priority == 0 && widget.DefaultPriority > 0 {
				req.Priority = widget.DefaultPriority
			}
			if len(req.Labels) == 0 && widget.DefaultLabels != "" {
				req.Labels = strings.Split(widget.DefaultLabels, ",")
				for i := range req.Labels {
					req.Labels[i] = strings.TrimSpace(req.Labels[i])
				}
			}

			task, err := widget.client.CreateTask(ctx, &req)
			if err != nil {
				slog.Error("Failed to create Todoist task", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			widget.scheduleEarlyUpdate()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		default:
			http.Error(w, "unknown action", http.StatusNotFound)
		}

	case http.MethodPut:
		// Update comment: /comments/{id}
		if strings.HasPrefix(path, "/comments/") {
			commentID := strings.TrimPrefix(path, "/comments/")
			commentID = strings.Split(commentID, "/")[0]

			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			comment, err := widget.client.UpdateCommentContent(ctx, commentID, req.Content)
			if err != nil {
				slog.Error("Failed to update Todoist comment", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(comment)
			return
		}

		// Update task: /tasks/{id} (not /close or /reopen)
		if strings.HasPrefix(path, "/tasks/") && !strings.HasSuffix(path, "/close") && !strings.HasSuffix(path, "/reopen") {
			taskID := strings.TrimPrefix(path, "/tasks/")
			taskID = strings.Split(taskID, "/")[0]

			var req todoist.UpdateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			task, err := widget.client.UpdateTask(ctx, taskID, &req)
			if err != nil {
				slog.Error("Failed to update Todoist task", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			widget.scheduleEarlyUpdate()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)
			return
		}
		http.Error(w, "unknown action", http.StatusNotFound)

	case http.MethodDelete:
		// Delete a task: /tasks/{id}
		taskPath := strings.TrimPrefix(path, "/tasks/")
		if taskPath == "" || taskPath == path {
			http.Error(w, "invalid task ID", http.StatusBadRequest)
			return
		}

		taskID := strings.Split(taskPath, "/")[0]
		if err := widget.client.DeleteTask(ctx, taskID); err != nil {
			slog.Error("Failed to delete Todoist task", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		widget.scheduleEarlyUpdate()
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper function to get color class for priority
func getPriorityColorClass(priority int) string {
	switch priority {
	case 4:
		return "priority-urgent"
	case 3:
		return "priority-high"
	case 2:
		return "priority-medium"
	default:
		return "priority-normal"
	}
}
