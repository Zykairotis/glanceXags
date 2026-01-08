# Bug: Todoist Widget Styling Improvements

## Bug Description
The Todoist widget has several visual styling issues that negatively impact user experience:
1. **Checkbox styling**: The task completion checkboxes use square styling instead of circular, which looks less polished
2. **Markdown rendering**: Comment content is displayed as plain text without markdown rendering, even though Todoist comments can contain markdown formatting
3. **Typography issues**: Font sizes, weights, and spacing throughout the widget are not optimized for readability
4. **Theme inconsistency**: The widget doesn't properly follow the Underworld (Red) theme color scheme

## Problem Statement
The Todoist widget's current styling lacks visual polish and readability. The checkbox uses a default browser square appearance instead of a custom circular design. Comments containing markdown are rendered as plain text, losing formatting. Typography choices are not optimized for the dark theme, making the widget content harder to read. The overall visual hierarchy needs improvement to match Glance's design standards.

## Solution Statement
1. Replace the default checkbox input with a custom circular checkbox design using CSS pseudo-elements
2. Implement markdown parsing and rendering for comment content using a lightweight markdown parser
3. Enhance typography with better font sizes, weights, line-heights, and spacing optimized for the Underworld Red theme
4. Ensure all colors properly use CSS custom properties from the theme system for consistency

## Steps to Reproduce
1. Navigate to the Todoist page in the Glance dashboard
2. Observe the square checkboxes next to each task
3. Click the comments button on any task with comments
4. Note that markdown formatting (bold, italic, links, code blocks) is not rendered
5. Compare the overall styling with the Underworld Red theme to see inconsistencies

## Root Cause Analysis
The issues stem from three main problems:

1. **Default checkbox styling**: The checkbox input uses the browser's default styling (square) instead of a custom circular design because the CSS only sets accent-color without customizing the appearance

2. **No markdown rendering**: The `appendComment` function in `todoist.js` (line 340-353) directly inserts escaped HTML content without parsing markdown formatting:
   ```javascript
   commentEl.innerHTML = `
       <div class="todoist-comment-content">${escapeHtml(comment.content)}</div>
   ```

3. **Typography not theme-optimized**: The CSS in `widget-todoist.css` uses generic font sizes and weights without considering the dark theme's contrast requirements and readability needs. The Underworld Red theme has `background-color: 0 0 3` (very dark) and `primary-color: 0 100 50` (pure red), requiring careful attention to text contrast.

## Relevant Files
Use these files to fix the bug:

### internal/glance/static/css/widget-todoist.css
- Contains all Todoist widget styling
- Needs custom circular checkbox implementation using `::before` and `::after` pseudo-elements
- Requires typography improvements for better readability
- Must ensure proper use of theme CSS custom properties

### internal/glance/static/js/todoist.js
- Contains the comment rendering logic in the `appendComment` function (lines 340-353)
- Needs markdown parsing implementation before rendering comment content
- Must preserve HTML escaping while allowing markdown-rendered content

### internal/glance/templates/todoist.html
- The template structure is fine, but verify the checkbox input has proper attributes for custom styling
- Comment content rendering is handled via JavaScript, so template changes are minimal

### New Files

None - we will implement markdown parsing using a lightweight, vanilla JavaScript approach without external dependencies to maintain Glance's minimal dependency philosophy.

## Step by Step Tasks
IMPORTANT: Execute every step in order, top to bottom.

### 1. Implement Circular Checkbox Styling
- Modify the `.todoist-checkbox` class in `widget-todoist.css`
- Hide the default checkbox input using `appearance: none` and `opacity: 0`
- Create a custom circular checkbox using `::before` pseudo-element with:
  - Circular shape using `border-radius: 50%`
  - Border using theme colors (var(--primary) for checked state)
  - Proper sizing (width and height matching current 1.25rem)
  - Center positioning
- Add checkmark icon using `::after` pseudo-element that appears when checkbox is checked
- Ensure proper hover and focus states for accessibility
- Test checkbox interaction and visual appearance

### 2. Implement Markdown Rendering for Comments
- Create a lightweight markdown parser function in `todoist.js` called `parseMarkdown(text)` that supports:
  - Bold text: `**text**` or `__text__`
  - Italic text: `*text*` or `_text_`
  - Links: `[text](url)`
  - Code blocks: `` `code` `` and multiline code blocks
  - Line breaks and paragraphs
- Modify the `appendComment` function to:
  - Parse the comment content with `parseMarkdown()` instead of just escaping HTML
  - Sanitize the output to prevent XSS (only allow safe markdown HTML tags)
  - Render the parsed markdown in the `.todoist-comment-content` element
- Ensure code blocks have proper styling (monospace font, background color)
- Add styles for markdown elements (links, code blocks, etc.) to `widget-todoist.css`

### 3. Enhance Typography for Underworld Red Theme
- Update `.todoist-task-title` with:
  - Font size: 0.95rem (slightly larger for better readability)
  - Font weight: 500 (medium weight for better hierarchy)
  - Line-height: 1.5 (improved readability)
  - Letter-spacing: 0.01em (slightly improved readability)
- Update `.todoist-comment-content` with:
  - Font size: 0.9rem
  - Line-height: 1.6 (better for longer text)
  - Proper spacing for markdown elements
- Update `.todoist-task-description` with:
  - Font size: 0.85rem (appropriate hierarchy below title)
  - Line-height: 1.5
  - Opacity: 0.85 (better contrast on dark background)
- Improve `.todoist-task-meta` typography:
  - Font size: 0.8rem (clear but not distracting)
  - Font weight: 500 (medium weight for labels)
  - Letter-spacing: 0.02em (improved readability for small text)
- Ensure all colors use theme CSS custom properties:
  - Use `var(--paragraph)` for body text
  - Use `var(--primary)` for links and interactive elements
  - Use `var(--negative)` for deadlines/errors
  - Maintain proper contrast ratios for accessibility

### 4. Improve Visual Hierarchy and Spacing
- Add subtle border or shadow to task cards for better separation:
  - Use `border: 1px solid rgba(255, 255, 255, 0.05)` for subtle definition
  - Keep the existing background color transition on hover
- Improve spacing between sections:
  - Increase gap in `.todoist-tasks` from 0.75rem to 1rem
  - Add more padding inside task cards (from 0.75rem to 1rem)
- Enhance comment section styling:
  - Add subtle background: `rgba(255, 255, 255, 0.03)`
  - Improve border styling with theme-aware colors
- Ensure priority indicators have better visual weight

### 5. Add Markdown Element Styling
- Add styles for rendered markdown elements:
  - `strong` and `b` tags: Font weight 600, use theme color
  - `em` and `i` tags: Font style italic
  - `a` tags: Use `var(--primary)` color, underline on hover
  - `code` tags: Monospace font, background `rgba(255, 255, 255, 0.1)`, padding 0.2rem 0.4rem
  - `pre` tags: Multiline code blocks with proper padding and overflow handling
  - `p` tags: Margin bottom 0.5rem for paragraph spacing
- Ensure all markdown styles respect the dark theme

### 6. Test and Validate
- Build the project: `go build -o build/glance .`
- Start the server: `./build/glance`
- Navigate to the Todoist page
- Verify checkbox appearance is circular and interactive
- Test markdown rendering in comments with various formatting:
  - Bold and italic text
  - Links
  - Code snippets
- Check typography improvements for readability
- Verify theme consistency with Underworld Red colors
- Test on different screen sizes (mobile responsive)
- Verify accessibility with keyboard navigation and screen readers

## Validation Commands
Execute every command to validate the bug is fixed with zero regressions.

- `go build -o build/glance .` - Build the binary to ensure no compilation errors
- `go test ./...` - Run all tests to ensure no regressions (minimal test coverage currently)
- `./build/glance config:validate` - Validate configuration files
- Manual testing: Start server and verify visually that checkboxes are circular, markdown renders correctly, and typography is improved

## Notes
- Glance has a philosophy of minimal dependencies, so we will implement markdown parsing in vanilla JavaScript rather than adding a library like `marked` or `markdown-it`
- The Underworld Red theme uses very dark background (HSL 0 0 3) and pure red primary color (HSL 0 100 50), so ensure all text has sufficient contrast
- When implementing markdown parsing, be extremely careful about XSS vulnerabilities - only allow a safe whitelist of HTML tags (`p`, `strong`, `em`, `a`, `code`, `pre`)
- The checkbox customization must maintain accessibility (proper ARIA labels, keyboard navigation, focus states)
- Consider adding a loading state while markdown is being parsed for very long comments
- Test the changes with the Underworld Red theme active to ensure proper color integration
