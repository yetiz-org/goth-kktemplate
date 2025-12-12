// kktemplate_test.go contains unit tests for the kktemplate package.
 //
 // Test Case Index:
 // - TestLoadHtml_FallbackToMainLanguage: LoadHtml falls back from a region tag (e.g. zh-TW) to its base language (zh).
 // - TestLoadHtml_FallbackToDefault: LoadHtml falls back to the "default" language when no matching language exists.
 // - TestLoadHtml_NotFound: LoadHtml returns ErrTemplateNotFound when the requested template does not exist.
 // - TestLoadHtml_Cache_NoDebug: LoadHtml caches templates when debug mode is off and ignores subsequent file changes.
 // - TestLoadHtml_Cache_Debug: LoadHtml reloads templates on each call when KKAPP_DEBUG is enabled.
 // - TestLoadHtml_FuncMap: LoadHtml applies the global FuncMap when parsing templates.
 // - TestLoadText_Basic: LoadText loads and executes a text template successfully.
 // - TestLoadText_NotFound: LoadText returns ErrTemplateNotFound when the requested template does not exist.
 // - TestLoadText_Cache_Debug: LoadText reloads templates on each call when KKAPP_DEBUG is enabled.
 // - TestLoadText_FuncMap: LoadText applies the global FuncMap when parsing templates.
 // - TestLoadFrameHtml_NotFound_WhenFrameMissing: LoadFrameHtml returns ErrTemplateNotFound if required frame templates are missing.
 // - TestLoadFrameHtml_Basic: LoadFrameHtml loads the page template with frame templates and executes the composed output.
package kktemplate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	html "html/template"
	text "text/template"
)

 // withTempTemplateRoot creates an isolated template root directory under t.TempDir().
 //
 // The returned path matches the package's expected on-disk layout:
 //   <temp>/resources/template
 // Tests use this helper to avoid coupling to real repository resources.
func withTempTemplateRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "resources", "template")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir temp template root: %v", err)
	}
	return root
}

 // writeTemplateFile writes a single template fixture to the temporary template tree.
 //
 // It creates <root>/<lang>/<name>.tmpl with the provided content and returns the full file path.
 // The helper fails the test immediately on any filesystem error.
func writeTemplateFile(t *testing.T, root, lang, name, content string) string {
	t.Helper()
	dir := filepath.Join(root, lang)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	path := filepath.Join(dir, name+".tmpl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write template file: %v", err)
	}
	return path
}

 // resetGlobals reinitializes package-level global state that affects template loading.
 //
 // The kktemplate loaders maintain caches and configuration in globals (e.g. TemplateRootPath,
 // template maps, and FuncMap). Tests must reset these between cases to prevent cross-test
 // contamination. This helper also registers a Cleanup to restore the previous state.
func resetGlobals(t *testing.T, newRoot string) {
	t.Helper()
	oldRoot := TemplateRootPath
	oldHTML := htmlTemplateMap
	oldFrameHTML := frameHtmlTemplateMap
	oldText := textTemplateMap
	oldFrameExist := frameExist
	oldFuncMap := FuncMap

	TemplateRootPath = newRoot
	htmlTemplateMap = map[string]*html.Template{}
	frameHtmlTemplateMap = map[string]*html.Template{}
	textTemplateMap = map[string]*text.Template{}
	frameExist = false
	FuncMap = html.FuncMap{}

	t.Cleanup(func() {
		TemplateRootPath = oldRoot
		htmlTemplateMap = oldHTML
		frameHtmlTemplateMap = oldFrameHTML
		textTemplateMap = oldText
		frameExist = oldFrameExist
		FuncMap = oldFuncMap
	})
}

 // TestLoadHtml_FallbackToMainLanguage verifies that LoadHtml falls back from a region-specific
 // language tag (e.g. "zh-TW") to its base language ("zh") when the region variant is not present.
 // The executed template output must match the base language template content.
func TestLoadHtml_FallbackToMainLanguage(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	writeTemplateFile(t, root, "zh", "hello", "zh")

	tmpl, err := LoadHtml("hello", "zh-TW")
	if err != nil {
		t.Fatalf("LoadHtml: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := buf.String(), "zh"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadHtml_FallbackToDefault verifies that LoadHtml falls back to the "default" language
 // when no template exists for the requested language (including its base language).
 // The executed template output must match the default template content.
func TestLoadHtml_FallbackToDefault(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	writeTemplateFile(t, root, "default", "hello", "default")

	tmpl, err := LoadHtml("hello", "fr-FR")
	if err != nil {
		t.Fatalf("LoadHtml: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := buf.String(), "default"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadHtml_NotFound verifies that LoadHtml returns ErrTemplateNotFound when no template
 // exists for the requested name across all fallback paths.
func TestLoadHtml_NotFound(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	_, err := LoadHtml("missing", "en-US")
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrTemplateNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

 // TestLoadHtml_Cache_NoDebug verifies that LoadHtml uses a cached parsed template when debug
 // mode is off, even if the underlying template file changes on disk.
 // The test asserts both pointer identity (same template instance) and output stability.
func TestLoadHtml_Cache_NoDebug(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	path := writeTemplateFile(t, root, "default", "hello", "v1")

	tmpl1, err := LoadHtml("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadHtml(v1): %v", err)
	}
	var buf1 bytes.Buffer
	if err := tmpl1.Execute(&buf1, nil); err != nil {
		t.Fatalf("Execute(v1): %v", err)
	}
	if got, want := buf1.String(), "v1"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}

	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("rewrite template: %v", err)
	}

	tmpl2, err := LoadHtml("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadHtml(v2): %v", err)
	}
	if tmpl1 != tmpl2 {
		t.Fatalf("expected cached template")
	}
	var buf2 bytes.Buffer
	if err := tmpl2.Execute(&buf2, nil); err != nil {
		t.Fatalf("Execute(v2): %v", err)
	}
	if got, want := buf2.String(), "v1"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadHtml_Cache_Debug verifies that LoadHtml reparses template files when KKAPP_DEBUG is
 // enabled. After rewriting the template file, the second load must reflect the new content.
func TestLoadHtml_Cache_Debug(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)
	t.Setenv("KKAPP_DEBUG", "TRUE")

	path := writeTemplateFile(t, root, "default", "hello", "v1")

	tmpl1, err := LoadHtml("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadHtml(v1): %v", err)
	}
	var buf1 bytes.Buffer
	if err := tmpl1.Execute(&buf1, nil); err != nil {
		t.Fatalf("Execute(v1): %v", err)
	}
	if got, want := buf1.String(), "v1"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}

	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("rewrite template: %v", err)
	}

	tmpl2, err := LoadHtml("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadHtml(v2): %v", err)
	}
	var buf2 bytes.Buffer
	if err := tmpl2.Execute(&buf2, nil); err != nil {
		t.Fatalf("Execute(v2): %v", err)
	}
	if got, want := buf2.String(), "v2"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadHtml_FuncMap verifies that the global FuncMap is applied to HTML templates.
 // The template calls a function from FuncMap and the execution output must match.
func TestLoadHtml_FuncMap(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	FuncMap = html.FuncMap{"X": func() string { return "OK" }}
	writeTemplateFile(t, root, "default", "hello", "{{X}}")

	tmpl, err := LoadHtml("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadHtml: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := buf.String(), "OK"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadText_Basic verifies that LoadText loads a text/template template and can execute it
 // without errors, producing the expected output.
func TestLoadText_Basic(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	writeTemplateFile(t, root, "default", "hello", "hi")

	tmpl, err := LoadText("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadText: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := buf.String(), "hi"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadText_NotFound verifies that LoadText returns ErrTemplateNotFound when the requested
 // template does not exist.
func TestLoadText_NotFound(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	_, err := LoadText("missing", "en-US")
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrTemplateNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

 // TestLoadText_Cache_Debug verifies that LoadText reparses text templates when KKAPP_DEBUG is
 // enabled, so changes on disk are reflected in subsequent loads.
func TestLoadText_Cache_Debug(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)
	t.Setenv("KKAPP_DEBUG", "TRUE")

	path := writeTemplateFile(t, root, "default", "hello", "v1")

	tmpl1, err := LoadText("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadText(v1): %v", err)
	}
	var buf1 bytes.Buffer
	if err := tmpl1.Execute(&buf1, nil); err != nil {
		t.Fatalf("Execute(v1): %v", err)
	}
	if got, want := buf1.String(), "v1"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}

	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("rewrite template: %v", err)
	}

	tmpl2, err := LoadText("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadText(v2): %v", err)
	}
	var buf2 bytes.Buffer
	if err := tmpl2.Execute(&buf2, nil); err != nil {
		t.Fatalf("Execute(v2): %v", err)
	}
	if got, want := buf2.String(), "v2"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadText_FuncMap verifies that the global FuncMap is applied to text templates.
 // Even though FuncMap is typed as html.FuncMap, it should still be usable for text/template parsing.
func TestLoadText_FuncMap(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	FuncMap = html.FuncMap{"X": func() string { return "OK" }}
	writeTemplateFile(t, root, "default", "hello", "{{X}}")

	tmpl, err := LoadText("hello", "en-US")
	if err != nil {
		t.Fatalf("LoadText: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := buf.String(), "OK"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}

 // TestLoadFrameHtml_NotFound_WhenFrameMissing verifies that LoadFrameHtml returns ErrTemplateNotFound
 // if the page template exists but required frame templates have not been provided.
func TestLoadFrameHtml_NotFound_WhenFrameMissing(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	writeTemplateFile(t, root, "default", "page", "page")

	_, err := LoadFrameHtml("page", "en-US")
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrTemplateNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

 // TestLoadFrameHtml_Basic verifies that LoadFrameHtml loads all required frame templates and the
 // requested page template, and that executing the page template renders a composed result.
 // The test uses StructTemplateFrames to generate the minimal set of required frame templates.
func TestLoadFrameHtml_Basic(t *testing.T) {
	root := withTempTemplateRoot(t)
	resetGlobals(t, root)

	for _, frame := range StructTemplateFrames {
		writeTemplateFile(t, root, "default", frame, frame)
	}

	pagePath := writeTemplateFile(t, root, "default", "page", "page->{{template \"_main.tmpl\"}}")

	tmpl, err := LoadFrameHtml("page", "en-US")
	if err != nil {
		t.Fatalf("LoadFrameHtml: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(pagePath), nil); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	if got, want := buf.String(), "page->_main"; got != want {
		t.Fatalf("output mismatch: got %q want %q", got, want)
	}
}
