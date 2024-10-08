package annotate

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/khulnasoft-lab/fastnode/fastnode-go/lang"
	"github.com/khulnasoft-lab/fastnode/fastnode-go/sandbox"
)

// A Flow represents a list of alternating code/output segments.
type Flow struct {
	Language   lang.Language
	Stencil    *Stencil
	Raw        *sandbox.Result
	Segments   []Segment
	InputFiles []*InputFile
}

// Segment represents one of the alternating code/output blocks making up a code example flow
type Segment interface {
	String() string
}

// OutputSegment is an interface for an output block generated by a code example
type OutputSegment interface {
	Segment
	LineNumber() int
}

// A PlaintextAnnotation is an output annotation to be displayed as a plain text blob
type PlaintextAnnotation struct {
	Line       int    `json:"line"`
	Expression string `json:"expression"`
	Value      string `json:"value"`
}

// LineNumber returns the line number for the segment
func (p *PlaintextAnnotation) LineNumber() int { return p.Line }

// String returns a textual representation of this segment
func (p *PlaintextAnnotation) String() string {
	s := p.Value
	if p.Expression != "" {
		s = p.Expression + " = " + s
	}
	return s
}

// DirEntry represents one file or folder within a directory.
type DirEntry struct {
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
	Size        int64  `json:"size"`
	Modified    int64  `json:"modified"`
	Created     int64  `json:"created"`
	Accessed    int64  `json:"accessed"`
	OwnerID     int64  `json:"ownerid"`
	Owner       string `json:"owner"`
	GroupID     int64  `json:"groupid"`
	Group       string `json:"group"`
}

// DirTableAnnotation is an output annotation containing information about the
// contents of a directory.
type DirTableAnnotation struct {
	Line    int        `json:"line"`
	Path    string     `json:"path"`
	Caption string     `json:"caption"`
	Cols    []string   `json:"cols"`
	Entries []DirEntry `json:"entries"`
}

// LineNumber returns the line number for the segment
func (p *DirTableAnnotation) LineNumber() int { return p.Line }

// String returns a textual representation of this segment
func (p *DirTableAnnotation) String() string {
	return fmt.Sprintf("[%s]", p.Path)
}

// DirTreeAnnotation is an output annotation containing a directory
// tree structure in a flat format.
type DirTreeAnnotation struct {
	Line    int               `json:"line"`
	Path    string            `json:"path"`
	Caption string            `json:"caption"`
	Entries map[string]string `json:"entries"`
}

// LineNumber returns the line number for the segment
func (p *DirTreeAnnotation) LineNumber() int { return p.Line }

// String returns a textual representation of this segment
func (p *DirTreeAnnotation) String() string {
	return fmt.Sprintf("[%s]", p.Path)
}

// An ImageAnnotation is an output annotation containing an encoded image
type ImageAnnotation struct {
	Line     int    `json:"line"`
	Path     string `json:"path"`
	Data     []byte `json:"data"`
	Encoding string `json:"encoding"`
	Caption  string `json:"caption"`
}

// LineNumber returns the line number for the segment
func (p *ImageAnnotation) LineNumber() int { return p.Line }

// String returns a textual representation of this segment
func (p *ImageAnnotation) String() string {
	return fmt.Sprintf("[%s]", p.Path)
}

// A FileAnnotation is an output annotation containing the contents of a file
type FileAnnotation struct {
	Line    int    `json:"line"`
	Path    string `json:"path"`
	Data    []byte `json:"data"`
	Caption string `json:"caption"`
}

// LineNumber returns the line number for the segment
func (p *FileAnnotation) LineNumber() int { return p.Line }

// String returns a textual representation of this segment
func (p *FileAnnotation) String() string {
	return fmt.Sprintf("[%s]", p.Path)
}

// A CodeSegment represents one or more lines of code.
type CodeSegment struct {
	BeginLineNumber int // Line of code (0-indexed) which begins this segment
	EndLineNumber   int // Line of code (0-indexed) which ends this segment
	Code            string
}

// String returns a textual representation of this segment
func (p *CodeSegment) String() string { return p.Code }

// Region represents a part of a code example (prelude, main, postlude)
type Region struct {
	Name string
	Code string
}

// A RegionDelimiter indicates the beginning of the prelude, main, or postlude region
type RegionDelimiter struct {
	Line   int    `json:"line"`
	Region string `json:"region"`
}

// LineNumber returns the line number for the segment
func (p *RegionDelimiter) LineNumber() int { return p.Line }

// String returns a textual representation of this segment
func (p *RegionDelimiter) String() string {
	return fmt.Sprintf("[%s]", p.Region)
}

// joinRegions joins multiple regions of code. Ensures that the joined block ends with a blank line.
func joinRegions(regions []Region, language lang.Language) string {
	var joined string
	for _, region := range regions {
		// Before appending a new region, make sure the `joined` ends with a blank line.
		if len(joined) > 0 && !strings.HasSuffix(joined, "\n\n") {
			if strings.HasSuffix(joined, "\n") {
				joined += "\n"
			} else {
				joined += "\n\n"
			}
		}
		switch language {
		case lang.Python:
			joined += fmt.Sprintf(`##fastnode.show_region_delimiter("%s")`, region.Name) + "\n"
		case lang.Bash:
			joined += fmt.Sprintf(`##fastnode_show_region_delimiter %s`, region.Name) + "\n"
		}
		joined += region.Code
	}
	return joined
}

// Options represents options for running code examples
type Options struct {
	// DockerImage is the name of the docker image to run the example in. Must be non-empty.
	DockerImage string
	// Language is the language for the code example
	Language lang.Language
	// Entrypoint is the python source to run as the entrypoint, or nil to use the default entrypoint.
	Entrypoint []byte
}

// Run takes a code string and executes it.
func Run(code string, opts Options) (*Flow, error) {
	region := Region{
		Name: "main",
		Code: code,
	}
	return RunWithRegions([]Region{region}, region.Code, opts)
}

// RunWithRegions executes the given code example and attempts to place annotations inline. If
// that fails, it drops back to out-of-line annotations.
func RunWithRegions(regions []Region, specRegion string, opts Options) (*Flow, error) {
	// Parse apparatus spec
	spec, err := NewSpecFromCode(opts.Language, specRegion)
	if err != nil {
		log.Println("Error parsing apparatus spec:", err)
		return nil, err
	}
	return RunWithSpec(regions, spec, opts)
}

// RunWithSpec executes the given code example with the given spec.
func RunWithSpec(regions []Region, spec *Spec, opts Options) (*Flow, error) {
	apparatus, err := spec.BuildApparatus()
	if err != nil {
		log.Println("Error constructing new apparatus from spec:", err)
		return nil, err
	}

	code := joinRegions(regions, lang.FromFilename(spec.SaveAs))
	code = RemoveSpecFromCode(opts.Language, code)

	// Run the code
	return runInline(code, apparatus, spec, opts)
}

// runInline executes the given code example and returns inline output annotations. If that fails then it returns an error
func runInline(code string, apparatus *sandbox.Apparatus, spec *Spec, opts Options) (*Flow, error) {
	if opts.DockerImage == "" {
		return nil, fmt.Errorf("running examples outside of docker is not supported")
	}

	// Parse the source
	source, err := ParseExample(code, lang.FromFilename(spec.SaveAs))
	if err != nil {
		return nil, fmt.Errorf("error parsing example source: %v", err)
	}

	// Construct program
	var program sandbox.Program
	switch opts.Language {
	case lang.Python:
		entrypoint := pythonPresentationEntrypoint
		if opts.Entrypoint != nil {
			entrypoint = opts.Entrypoint
		}
		prog := sandbox.NewContainerizedPythonProgram(string(entrypoint), opts.DockerImage)
		prog.SupportingFiles["fastnode.py"] = pythonPresentationAPI
		prog.SupportingFiles[spec.SaveAs] = []byte(source.Runnable)
		prog.EnvironmentVariables["SOURCE"] = spec.SaveAs
		for k, v := range spec.Env {
			prog.EnvironmentVariables[k] = v
		}
		program = prog
	case lang.Bash:
		program = sandbox.NewContainerizedBashProgram(source.Runnable, opts.DockerImage)
	default:
		return nil, fmt.Errorf("unknown language: %s", opts.Language.Name())
	}

	// Run the apparatus
	result, err := apparatus.Run(program)
	if err != nil {
		return nil, err
	}

	// Get the annotations
	annotations, err := loadAnnotations(result)
	if err != nil {
		return nil, err
	}

	// Make a list of files for which we already have annotations
	usedFiles := make(map[string]struct{})
	for _, annotation := range annotations {
		switch annotation := annotation.(type) {
		case *ImageAnnotation:
			usedFiles[annotation.Path] = struct{}{}
		case *FileAnnotation:
			usedFiles[annotation.Path] = struct{}{}
		}
	}

	var segments []Segment
	err = nil
	if spec.Inline {
		segments, err = processInline(source, result, annotations)
	}
	if err != nil || !spec.Inline {
		segments = annotateOutOfLine(source.Presentation, result, annotations)
	}

	// Construct the output
	flow := &Flow{
		Language:   opts.Language,
		Stencil:    source,
		Raw:        result,
		Segments:   segments,
		InputFiles: spec.InputFiles,
	}

	return flow, nil
}

// annotateOutOfLine generates a single code segment followed by a single output segment containing
// all standard output
func annotateOutOfLine(source string, output *sandbox.Result, annotations []OutputSegment) []Segment {
	var lines []string
	if len(output.Stderr) > 0 {
		lines = append(lines, string(output.Stderr))
	}

	// Append HTTP outputs
	for _, r := range output.HTTPOutputs {
		if r.Response != nil {
			// Make sure there is an empty line before each HTTP response
			if len(lines) > 0 {
				lines = append(lines, "")
			}

			lines = append(lines, r.Response.Status)
			for key, vals := range r.Response.Header {
				lines = append(lines, key+": "+strings.Join(vals, ","))
			}
			if len(r.ResponseBody) > 0 {
				lines = append(lines, "\n"+string(r.ResponseBody))
			}
		}
	}

	// Append error message
	if output.SandboxError != nil {
		// Make sure there is an empty line before the error message
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, output.SandboxError.Error())
	}

	// Insert code segment
	segments := []Segment{&CodeSegment{
		BeginLineNumber: 0,
		EndLineNumber:   strings.Count(source, "\n"),
		Code:            source,
	}}

	// Insert stdout if there is any
	stdout := strings.Join(lines, "\n")
	if strings.TrimSpace(stdout) != "" {
		segments = append(segments, &PlaintextAnnotation{
			Value: stdout,
		})
	}

	// Then append any other annotations
	for _, annotation := range annotations {
		segments = append(segments, annotation)
	}

	return segments
}

func processInline(source *Stencil, result *sandbox.Result, annotations []OutputSegment) ([]Segment, error) {
	if !source.Inline {
		return nil, errors.New("code example could not be parsed for inline annotations")
	}
	if !result.Succeeded {
		return nil, errors.New("code example exited with nonzero exit status:\n" + string(result.Stderr))
	}
	if len(result.Stderr) > 0 {
		return nil, errors.New("code example wrote to standard error:\n" + string(result.Stderr))
	}
	if len(result.HTTPOutputs) > 0 {
		return nil, errors.New("code example generated an HTTP output")
	}

	// Interleave annotations with code
	segments, err := annotateInline(source.Presentation, source.LineMap, annotations)
	if err != nil {
		return nil, err
	}
	return segments, nil
}

// AnnotateInline merges output segments from a standard output stream with a parsed code example and
// returns a list of interleaved code/output flow.Segments
func annotateInline(source string, lineMap []int, annotations []OutputSegment) ([]Segment, error) {
	lines := strings.Split(source, "\n")
	if lineMap == nil {
		for i := range lines {
			lineMap = append(lineMap, i)
		}
	}

	// Parse the annotations line by line
	var prevline, lineMapIndex int
	var segments []Segment
	for _, annotation := range annotations {
		// Check that the annotations are in order
		line := annotation.LineNumber()
		if line < prevline {
			return nil, fmt.Errorf("line numbers on annotations were out of order: %d followed by %d", prevline, line)
		}

		// Determine how many new lines of code to add
		// Remember: lineMap maps from original source line to annotated source line
		var code []string
		beginLineNumber := lineMapIndex
		for lineMapIndex < len(lineMap) && lineMap[lineMapIndex] <= line {
			code = append(code, lines[lineMapIndex])
			lineMapIndex++
		}
		joined := strings.Join(code, "\n")

		// Add the code segment, if not empty
		if len(strings.TrimSpace(joined)) > 0 {
			segments = append(segments, &CodeSegment{
				BeginLineNumber: beginLineNumber,
				EndLineNumber:   beginLineNumber + len(code) - 1,
				Code:            joined,
			})
		}

		// Add the output annotation
		segments = append(segments, annotation)
		prevline = line
	}

	if prevline+1 < len(lines) {
		code := strings.Join(lines[prevline+1:], "\n")
		if len(strings.TrimSpace(code)) > 0 {
			codesegment := &CodeSegment{
				BeginLineNumber: prevline + 1,
				EndLineNumber:   len(lines) - 1,
				Code:            code,
			}
			segments = append(segments, codesegment)
		}
	}

	return segments, nil
}

func parseAnnotation(buf []byte, line int) (OutputSegment, error) {
	// First discover the type (which must be present in all annotations)
	var f struct {
		Type string `json:"type"`
	}
	err := json.Unmarshal(buf, &f)
	if err != nil {
		return nil, fmt.Errorf("error decoding annotation: %v", err)
	}

	// Now decode the type-specific values
	var annotation OutputSegment
	switch f.Type {
	case "plaintext":
		annotation = &PlaintextAnnotation{Line: line}
	case "dirtable":
		annotation = &DirTableAnnotation{Line: line}
	case "dirtree":
		annotation = &DirTreeAnnotation{Line: line}
	case "image":
		annotation = &ImageAnnotation{Line: line}
	case "file":
		annotation = &FileAnnotation{Line: line}
	case "region":
		annotation = &RegionDelimiter{Line: line}
	default:
		return nil, fmt.Errorf("unrecognized annotation type: '%s'", f.Type)
	}

	err = json.Unmarshal(buf, annotation)
	if err != nil {
		return nil, fmt.Errorf("error decoding annotation of type %s into %T: %v", f.Type, annotation, err)
	}

	return annotation, nil
}

func loadAnnotations(output *sandbox.Result) ([]OutputSegment, error) {
	// Extract blobs from the raw output
	blobs, err := parseBlobs(string(output.Stdout))
	if err != nil {
		fmt.Println(string(output.Stdout))
		return nil, fmt.Errorf("error parsing blobs from code example output: %v", err)
	}

	// Parse the annotations line by line
	var line int
	var annotations []OutputSegment
	for _, blob := range blobs {
		switch blob.Type {
		case lineBlob:
			line = blob.Line
		case outputBlob:
			annotations = append(annotations, &PlaintextAnnotation{
				Line:  line,
				Value: blob.Content,
			})
		case emitBlob:
			annotation, err := parseAnnotation([]byte(blob.Content), line)
			if err != nil {
				return nil, err
			}
			annotations = append(annotations, annotation)
		}
	}

	return annotations, nil
}

// Plain renders just the plaintext output
func (ex *Flow) Plain() string {
	var s string
	for _, segment := range ex.Segments {
		if segment, ok := segment.(*PlaintextAnnotation); ok {
			s += segment.Value
			if !strings.HasSuffix(s, "\n") {
				s += "\n"
			}
		}
	}
	return s
}
