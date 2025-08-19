# aspec CLI

**aspec** is a Go-based CLI that transforms artifact/extractor JSON Schemas from `o3-cloud/artifact-specs` into:

- A plain-English prompt
- A JSON extraction from unstructured input (via LLM, with optional validation)
- A plain-English Markdown rendering (via JSON → LLM verbalization)

LLM calls are routed through OpenRouter using the OpenAI Go client.

## Installation

```bash
# Clone the repository and build
git clone https://github.com/o3-cloud/artifact-specs
cd artifact-specs/cli
go build -o bin/aspec ./cmd/aspec

# Add to PATH or install globally
go install ./cmd/aspec
```

## Configuration

### Initial Setup

```bash
# Create default configuration
aspec init

# Set your OpenRouter API key
export OPENROUTER_API_KEY=your_key_here
```

Configuration is stored in `~/.artifactspecs/config.yaml`:

```yaml
model: openai/gpt-4o-mini
base_url: https://openrouter.ai/api/v1
provider: openrouter
system_prompt_file: ""

# Customize prompts used for extraction and verbalization
prompts:
  extraction: |
    Extract structured data from the following input according to the "{{.SchemaTitle}}" specification.
    
    Instructions:
    - Extract only information that is explicitly present in the input
    - Do not invent or infer information not directly stated
    - Leave fields empty/null if the information is not available
    - Follow the JSON schema structure exactly
    - Ensure all required fields are present
    
    Input:
    {{.Input}}
    
    JSON Schema Reference:
    {{.Schema}}
    
    Provide the extracted data as valid JSON:
    
  verbalization: |
    Convert the following structured JSON data into clear, readable Markdown format.
    
    Requirements:
    - Use appropriate Markdown formatting (headers, lists, tables, etc.)
    - Present the information in a logical, easy-to-read structure
    - Include only the information present in the JSON data
    - Do not add commentary or extra headings beyond what the data implies
    - Maintain accuracy to the original data
    
    JSON Data:
    {{.JSONData}}
    
    Generate well-formatted Markdown:
```

### Prompt Customization

The CLI uses Go template syntax for prompt templates. You can customize the prompts by editing your config file:

- **Extraction prompt**: Uses variables `{{.SchemaTitle}}`, `{{.Input}}`, and `{{.Schema}}`
- **Verbalization prompt**: Uses variable `{{.JSONData}}`

Copy the example config (`config.example.yaml`) to `~/.artifactspecs/config.yaml` and modify the prompts to suit your needs.

### Environment Variables

- `OPENROUTER_API_KEY` (required) - Your OpenRouter API key
- `ASPEC_MODEL` - Override default model
- `ASPEC_PROVIDER_BASE_URL` - Override provider base URL

## Quick Start

### 1. Update Local Cache

First, cache the schemas from GitHub:

```bash
aspec update
```

### 2. List Available Specs

```bash
# List all artifacts
aspec list

# List extractors
aspec list --type extractors

# Search for specific specs
aspec list --search meeting
```

### 3. Extract Structured Data

```bash
# Extract from file (validation skipped by default for speed)
aspec extract --spec meeting_summary --in notes.md

# Extract with validation enabled
aspec extract --spec meeting_summary --in notes.md --validate

# Extract from directory (concatenates all files)  
aspec extract --spec meeting_summary --in ./documents/

# Extract from stdin
cat notes.md | aspec extract --spec meeting_summary
```

### 4. Render to Markdown

```bash
# Two-step: extract then verbalize to Markdown
aspec render --spec meeting_summary --in notes.md --out report.md

# Save intermediate JSON
aspec render --spec meeting_summary --in notes.md --out report.md --save-json
```

## Commands Reference

### `aspec list`

List available specs from local cache.

```bash
aspec list [flags]

Flags:
  --json              Output as JSON
  --yaml              Output as YAML  
  --search string     Filter specs by search term
  --type string       Spec type: artifacts or extractors (default "artifacts")
```

### `aspec show`

Print resolved raw schema.

```bash
aspec show [flags]

Flags:
  --spec string       Spec slug to show
  --type string       Spec type: artifacts or extractors (default "artifacts")
  --spec-url string   Spec URL to show
  --spec-path string  Spec file path to show
  --format string     Output format: json or yaml (default "json")
  --ref string        Git reference (default "main")
```

### `aspec prompt`

Turn a spec into a plain-English prompt.

```bash
aspec prompt [flags]

Flags:
  --spec string       Spec slug
  --type string       Spec type: artifacts or extractors (default "artifacts")
  --spec-url string   Spec URL
  --spec-path string  Spec file path
  --out string        Output file
  --model string      LLM model to use
  --stream            Stream output (default true)
  --no-stream         Disable streaming
```

### `aspec extract`

Extract structured JSON from unstructured input. **Automatically chunks large inputs** that exceed token limits.

```bash
aspec extract [flags]

Flags:
  --spec string                     Spec slug
  --type string                     Spec type: artifacts or extractors (default "artifacts")
  --spec-url string                 Spec URL
  --spec-path string                Spec file path
  --in string                       Input file or directory
  --out string                      Output file
  --model string                    LLM model to use
  --max-retries int                 Maximum retry attempts (default 2)
  --no-validate                     Skip validation (default true)
  --validate                        Enable validation (overrides --no-validate)
  --compact                         Compact JSON output
  --stats                           Show stats
  --no-stream                       Disable streaming (default true)
  --chunk-size int                  Maximum tokens per chunk for large inputs (default 20000)
  --merge-strategy string           Merge strategy: incremental, two-pass, template-driven (default "incremental")
  --merge-instructions string       Custom instructions for merging chunks
```

### `aspec render`

Render unstructured input to Markdown. **Automatically chunks large inputs** that exceed token limits.

```bash
aspec render [flags]

Flags:
  --spec string                     Spec slug
  --type string                     Spec type: artifacts or extractors (default "artifacts") 
  --spec-url string                 Spec URL
  --spec-path string                Spec file path
  --in string                       Input file or directory
  --out string                      Output file
  --model string                    LLM model to use
  --save-json                       Save intermediate JSON
  --stats                           Show stats
  --stream                          Stream output (default true)
  --no-stream                       Disable streaming
  --no-validate                     Skip validation (default true)
  --validate                        Enable validation (overrides --no-validate)
  --chunk-size int                  Maximum tokens per chunk for large inputs (default 20000)
  --merge-strategy string           Merge strategy: incremental, two-pass, template-driven (default "incremental")
  --merge-instructions string       Custom instructions for merging chunks
```

### `aspec validate`

Validate JSON against a spec.

```bash
aspec validate <json-file> [flags]

Flags:
  --spec string       Spec slug
  --type string       Spec type: artifacts or extractors (default "artifacts")
  --spec-url string   Spec URL  
  --spec-path string  Spec file path
```

### `aspec update`

Refresh local cache from GitHub.

```bash
aspec update [flags]

Flags:
  --ref string        Git reference (default "main")
  --repo string       Repository (default "o3-cloud/artifact-specs")
```

### `aspec test`

Run deterministic tests using mock provider.

```bash
aspec test [flags]

Flags:
  --spec string           Spec slug
  --fixture string        Fixture name
  --expected string       Expected output file
  --provider string       Provider to use (default "mock")
  --mock-fixture string   Mock response fixture
```

### Global Flags

```bash
  -v, --verbose       Verbose output (repeatable)
      --quiet         Quiet mode
      --log-json      Output logs as JSON
```

## Examples

### Basic Workflow

```bash
# Initialize and update
aspec init
aspec update

# Extract meeting summary from notes (fast, no validation)
aspec extract --spec meeting_summary --in meeting-notes.md --out summary.json --stats

# Extract with validation enabled
aspec extract --spec meeting_summary --in meeting-notes.md --out summary.json --validate --stats

# Render to readable Markdown
aspec render --spec meeting_summary --in meeting-notes.md --out report.md --save-json

# Validate existing JSON
aspec validate existing-summary.json --spec meeting_summary
```

### Working with Different Input Sources

```bash
# From file
aspec extract --spec prd --in requirements.md

# From directory (concatenates all files)
aspec extract --spec prd --in ./project-docs/

# From stdin
cat requirements.md | aspec extract --spec prd

# Using custom schema file
aspec extract --spec-path ./custom-schema.json --in data.txt
```

### Advanced Usage

```bash
# Use different model
aspec extract --spec meeting_summary --in notes.md --model anthropic/claude-3-haiku

# Enable validation (disabled by default for speed)
aspec extract --spec meeting_summary --in notes.md --validate

# Compact JSON output
aspec extract --spec meeting_summary --in notes.md --compact

# Show token usage and cost estimates
aspec extract --spec meeting_summary --in notes.md --stats

# Large document processing with chunking
aspec extract --spec prd --in huge-requirements.md --chunk-size 15000 --stats -v

# Custom merge strategy with instructions
aspec extract --spec meeting_summary --in transcript.txt \
  --merge-strategy two-pass \
  --merge-instructions "Combine duplicate action items, preserve all speaker names"

# Render large document with chunking
aspec render --spec meeting_summary --in large-transcript.txt --out report.md --chunk-size 15000

# Render with custom merge strategy  
aspec render --spec prd --in massive-requirements.md --out prd-report.md \
  --merge-strategy template-driven \
  --merge-instructions "Prioritize requirements by priority level, group related features"
```

## Exit Codes

- `0`: Success
- `1`: Invalid arguments/input
- `2`: LLM/API error  
- `3`: Schema validation failed
- `4`: Output write error

## Testing

```bash
# Run tests with mock provider
aspec test --spec meeting_summary \
  --fixture testdata/meeting_summary/input.md \
  --expected testdata/meeting_summary/expected.json \
  --provider mock \
  --mock-fixture testdata/meeting_summary/mock_response.json

# Build and test
make test
```

## FAQs

### How do I get an OpenRouter API key?

Visit [openrouter.ai](https://openrouter.ai) and create an account to get your API key.

### What models are supported?

Any model available through OpenRouter, including:
- `openai/gpt-4o-mini` (default)
- `openai/gpt-4o`
- `anthropic/claude-3-haiku`
- `anthropic/claude-3-sonnet`

### Can I use local schemas?

Yes, use `--spec-path` to point to local JSON schema files.

### Why is validation disabled by default?

Validation is disabled by default to improve performance and reduce LLM costs. The CLI prioritizes speed for most use cases. When validation is needed for accuracy or compliance, use the `--validate` flag to enable it.

### How does binary detection work?

The CLI automatically skips common binary file types (PDF, images, archives, executables) when processing directories, with warnings logged to stderr.

### When should I use chunking?

Chunking is automatically enabled when your input exceeds the `--chunk-size` limit (default 20,000 tokens). You might want to adjust chunk size or strategy for:

- **Very large documents** (100k+ tokens): Use smaller chunk sizes (10k-15k)
- **Independent sections**: Use `two-pass` strategy
- **Sequential content**: Use `incremental` strategy (default)
- **Complex schemas**: Use `template-driven` strategy

### How accurate is chunking vs single-pass?

Chunking maintains high accuracy by:
- Including the full JSON schema in every merge prompt
- Using semantic boundaries to preserve context
- Allowing custom merge instructions for domain-specific logic
- Supporting validation of the final merged result

### How are costs calculated?

Cost estimates are approximate based on known OpenRouter pricing and may vary. Use `--stats` to see token usage and rough cost estimates.

## Large Document Processing (Chunking)

The `aspec extract` and `aspec render` commands automatically handle large documents that exceed LLM context windows by intelligently chunking the input and merging the results.

### How It Works

1. **Automatic Detection**: When input exceeds `--chunk-size` tokens (default 20,000), chunking is automatically enabled
2. **Smart Splitting**: Text is split on semantic boundaries (paragraphs → sentences → words) for better context preservation
3. **Merge Strategies**: Three different approaches for combining chunk results
4. **Progress Tracking**: Detailed logging shows chunking and processing progress

### Chunking Flags

- `--chunk-size int`: Maximum tokens per chunk (default: 20000)
- `--merge-strategy string`: How to combine chunk results
- `--merge-instructions string`: Custom guidance for merging

### Merge Strategies

#### Incremental (Default)
Processes chunks sequentially, merging each new chunk with the accumulated result.

```bash
aspec extract --spec meeting_summary --in large-document.md --merge-strategy incremental
```

Best for: Most use cases, memory efficient, good for sequential content

#### Two-Pass
Processes all chunks independently, then merges all results together.

```bash
aspec extract --spec meeting_summary --in large-document.md --merge-strategy two-pass
```

Best for: When chunks are independent, parallel-friendly content

#### Template-Driven
Uses JSON schema structure to intelligently merge chunk results.

```bash
aspec extract --spec meeting_summary --in large-document.md --merge-strategy template-driven
```

Best for: Complex schemas with specific merge requirements

### Custom Merge Instructions

You can provide specific instructions for how chunks should be merged:

```bash
aspec extract --spec meeting_summary --in large-document.md \
  --merge-instructions "Prioritize newer information over older, combine action items into single list"
```

### Chunking Examples

```bash
# Large document with default chunking
aspec extract --spec prd --in massive-requirements.md --stats

# Custom chunk size for smaller context windows
aspec extract --spec meeting_summary --in long-transcript.txt --chunk-size 15000

# Two-pass strategy with validation
aspec extract --spec risk_summary --in reports/ --merge-strategy two-pass --validate

# Custom merge instructions
aspec extract --spec action_items --in project-files/ \
  --merge-instructions "Deduplicate similar tasks, group by priority level"

# Render large documents with chunking
aspec render --spec meeting_summary --in huge-transcript.txt --out summary-report.md --stats

# Render with custom chunk size and strategy
aspec render --spec prd --in requirements/ --out prd-document.md --chunk-size 15000 --merge-strategy two-pass
```

### Progress Monitoring

Use verbose flags to see detailed chunking progress:

```bash
# Basic progress information
aspec extract --spec summary --in large-doc.md -v

# Detailed chunk information
aspec extract --spec summary --in large-doc.md -vv
```

The chunking feature makes it possible to process arbitrarily large documents while maintaining the quality and structure of the extracted data.

## Exit Codes

- `0`: Success
- `1`: Invalid arguments/input
- `2`: LLM/API error
- `3`: Schema validation failed
- `4`: Output write error

## Testing

```bash
# Run tests with mock provider
aspec test --spec meeting_summary \
  --fixture testdata/meeting_summary/input.md \
  --expected testdata/meeting_summary/expected.json \
  --provider mock \
  --mock-fixture testdata/meeting_summary/mock_response.json

# Build and test
make test
```

## FAQs

### How do I get an OpenRouter API key?

Visit [openrouter.ai](https://openrouter.ai) and create an account to get your API key.

### What models are supported?

Any model available through OpenRouter, including:
- `openai/gpt-4o-mini` (default)
- `openai/gpt-4o`
- `anthropic/claude-3-haiku`
- `anthropic/claude-3-sonnet`

### Can I use local schemas?

Yes, use `--spec-path` to point to local JSON schema files.

### Why is validation disabled by default?

Validation is disabled by default to improve performance and reduce LLM costs. The CLI prioritizes speed for most use cases. When validation is needed for accuracy or compliance, use the `--validate` flag to enable it.

### How does binary detection work?

The CLI automatically skips common binary file types (PDF, images, archives, executables) when processing directories, with warnings logged to stderr.

### When should I use chunking?

Chunking is automatically enabled when your input exceeds the `--chunk-size` limit (default 20,000 tokens). You might want to adjust chunk size or strategy for:

- **Very large documents** (100k+ tokens): Use smaller chunk sizes (10k-15k)
- **Independent sections**: Use `two-pass` strategy
- **Sequential content**: Use `incremental` strategy (default)
- **Complex schemas**: Use `template-driven` strategy

### How accurate is chunking vs single-pass?

Chunking maintains high accuracy by:
- Including the full JSON schema in every merge prompt
- Using semantic boundaries to preserve context
- Allowing custom merge instructions for domain-specific logic
- Supporting validation of the final merged result

### How are costs calculated?

Cost estimates are approximate based on known OpenRouter pricing and may vary. Use `--stats` to see token usage and rough cost estimates.