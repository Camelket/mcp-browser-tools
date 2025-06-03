<#
.SYNOPSIS
    Build Go binaries, commit/tag, push, and create a GitHub Release with those binaries.

.DESCRIPTION
    This script automates the release process for a Go-based project. Given a version string (e.g. "1.0.0" or "v1.0.0"), it:
      1. Determines the Git tag (prepends “v” if missing).
      2. Builds platform-specific binaries (linux/amd64, darwin/amd64, windows/amd64).
      3. Commits any outstanding changes (if present) with a “Release <tag>” message.
      4. Creates an annotated Git tag.
      5. Pushes the current branch and the new tag to origin.
      6. Uses the GitHub CLI (`gh`) to create a Release with those binaries as assets.

.PARAMETER Version
    The version to release. May be prefixed with “v” or not (e.g. “1.2.3” or “v1.2.3”). 
    The script will normalize to “v<major>.<minor>.<patch>”.

.EXAMPLE
    PS> .\release.ps1 1.0.0

    Builds binaries for version “1.0.0”, creates tag “v1.0.0”, commits changes, pushes, 
    and publishes a GitHub Release with the generated binaries.

.NOTES
    - Requires:
        • Go installed and on PATH.
        • Git installed and on PATH.
        • GitHub CLI (`gh`) installed, authenticated, and on PATH.
        • `bin/` should be in `.gitignore`, so that built executables are not committed.
    - Run this script from the root of your Git repository.
#>

param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Version
)

function Throw-IfNotInstalled {
    param(
        [Parameter(Mandatory = $true)]
        [string]$CmdName
    )
    if (-not (Get-Command $CmdName -ErrorAction SilentlyContinue)) {
        Write-Error "‘$CmdName’ is not installed or not on PATH. Aborting."
        exit 1
    }
}

# Ensure required tools are present
Throw-IfNotInstalled -CmdName 'git'
Throw-IfNotInstalled -CmdName 'go'
Throw-IfNotInstalled -CmdName 'gh'

# Normalize version/tag
if ($Version.StartsWith('v')) {
    $tag = $Version
    $verNoV = $Version.Substring(1)
} else {
    $tag = "v$Version"
    $verNoV = $Version
}

$binaryBaseName = "mcp-browser-tools"

Write-Host "Releasing version: $verNoV (Git tag: $tag)"

# Verify we are inside a Git repository
$insideRepo = git rev-parse --is-inside-work-tree 2>$null
if ($LASTEXITCODE -ne 0 -or $insideRepo.Trim() -ne 'true') {
    Write-Error "Current directory is not inside a Git repository. Aborting."
    exit 1
}

# Determine current branch
$branch = (git rev-parse --abbrev-ref HEAD).Trim()
Write-Host "Current Git branch: $branch"

# Create/clean the bin directory
$binDir = Join-Path -Path (Get-Location) -ChildPath "bin"
if (Test-Path $binDir) {
    Remove-Item -Recurse -Force $binDir
}
New-Item -ItemType Directory -Path $binDir | Out-Null

# Helper to build a Go binary for a given OS/ARCH
function Build-GoBinary {
    param(
        [Parameter(Mandatory=$true)] [string]$GOOS,
        [Parameter(Mandatory=$true)] [string]$GOARCH,
        [Parameter(Mandatory=$true)] [string]$OutputPath
    )
    Write-Host "Building for $GOOS/$GOARCH → $OutputPath"
    $env:GOOS   = $GOOS
    $env:GOARCH = $GOARCH

    # If Windows target, append .exe
    if ($GOOS -eq 'windows' -and -not $OutputPath.EndsWith(".exe")) {
        $OutputPath = "$OutputPath.exe"
    }

    & go build -o $OutputPath ./main.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go build failed for $GOOS/$GOARCH. Aborting."
        exit 1
    }
}



# Build binaries
$linuxOut  = Join-Path $binDir "${binaryBaseName}_${verNoV}_linux_amd64"
$darwinOut = Join-Path $binDir "${binaryBaseName}_${verNoV}_darwin_amd64"
$windowsOut= Join-Path $binDir "${binaryBaseName}_${verNoV}_windows_amd64.exe"

Build-GoBinary -GOOS 'linux'   -GOARCH 'amd64' -OutputPath $linuxOut
Build-GoBinary -GOOS 'darwin'  -GOARCH 'amd64' -OutputPath $darwinOut
Build-GoBinary -GOOS 'windows' -GOARCH 'amd64' -OutputPath $windowsOut

# Restore environment variables (optional)
Remove-Item Env:GOOS   -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

# Check for uncommitted changes
$changes = git status --porcelain
if ($changes) {
    Write-Host "Uncommitted changes detected. Staging and committing..."
    git add .
    git commit -m "Release $tag"
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Git commit failed. Aborting."
        exit 1
    }
} else {
    Write-Host "No changes to commit."
}

# Create an annotated tag
Write-Host "Creating Git tag $tag"
git tag -a $tag -m "Release $tag"
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to create Git tag. Aborting."
    exit 1
}

# Push branch and tag
Write-Host "Pushing branch ‘$branch’ to origin"
git push origin $branch
if ($LASTEXITCODE -ne 0) {
    Write-Error "git push origin $branch failed. Aborting."
    exit 1
}

Write-Host "Pushing tag $tag to origin"
git push origin $tag
if ($LASTEXITCODE -ne 0) {
    Write-Error "git push origin $tag failed. Aborting."
    exit 1
}

# Ensure built binaries exist before releasing
$assets = @()
foreach ($path in @($linuxOut, $darwinOut, $windowsOut)) {
    if (-not (Test-Path $path)) {
        Write-Error "Expected asset not found: $path. Aborting release creation."
        exit 1
    }
    $assets += $path
}

# Create GitHub Release using gh CLI
Write-Host "Creating GitHub Release $tag with attached binaries..."
gh release create $tag `
    --title "$tag" `
    --notes "Release $tag" `
    $assets
if ($LASTEXITCODE -ne 0) {
    Write-Error "gh release create failed. Aborting."
    exit 1
}

Write-Host "Release $tag published successfully!"
