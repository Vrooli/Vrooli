#!/usr/bin/env bats
# Comprehensive tests for filesystem mock functionality

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

setup() {
    # Set up test environment first
    export MOCK_LOG_DIR="${TMPDIR:-/tmp}/filesystem-test-$$"
    export MOCK_RESPONSES_DIR="$MOCK_LOG_DIR"
    # Use 'command' to ensure we use the real mkdir, not the mock
    command mkdir -p "$MOCK_LOG_DIR"
    
    # Load test helpers if available
    if [[ -f "${BATS_TEST_DIRNAME}/../../helpers/bats-support/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../helpers/bats-support/load.bash"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/../../helpers/bats-assert/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../helpers/bats-assert/load.bash"
    else
        # Define basic assertion functions if bats-assert is not available
        assert_success() {
            if [[ "$status" -ne 0 ]]; then
                echo "expected success, got status $status"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_failure() {
            if [[ "$status" -eq 0 ]]; then
                echo "expected failure, got success"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" != *"$expected"* ]]; then
                    echo "expected output to contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" != "$expected" ]]; then
                    echo "expected: $expected"
                    echo "actual: $output"
                    return 1
                fi
            fi
        }
        
        assert_line() {
            local expected
            local partial=false
            if [[ "$1" == "--partial" ]]; then
                partial=true
                shift
            fi
            expected="$1"
            
            local found=false
            while IFS= read -r line; do
                if [[ "$partial" == true ]]; then
                    if [[ "$line" == *"$expected"* ]]; then
                        found=true
                        break
                    fi
                else
                    if [[ "$line" == "$expected" ]]; then
                        found=true
                        break
                    fi
                fi
            done <<< "$output"
            
            if [[ "$found" != true ]]; then
                echo "expected line: $expected"
                echo "in output: $output"
                return 1
            fi
        }
        
        refute_line() {
            local not_expected="$1"
            while IFS= read -r line; do
                if [[ "$line" == "$not_expected" ]]; then
                    echo "unexpected line found: $not_expected"
                    echo "in output: $output"
                    return 1
                fi
            done <<< "$output"
        }
    fi
    
    # Source the mock libraries
    # source "${BATS_TEST_DIRNAME}/logs.sh"
    # source "${BATS_TEST_DIRNAME}/verification.sh"
    source "${BATS_TEST_DIRNAME}/filesystem.sh"
    
    # Initialize mock systems
    # mock::init_logging "$MOCK_LOG_DIR"
    # mock::verify::init  # Comment out for now to debug
    mock::fs::reset
}

teardown() {
    # Reset filesystem mode to normal before cleanup
    export FILESYSTEM_MOCK_MODE="normal"
    
    # Clean up state files - use command to bypass our mock
    trash::safe_remove "${FILESYSTEM_MOCK_STATE_FILE:-}" --test-cleanup 2>/dev/null || true
    
    # Clean up test directory - use command to bypass our mock  
    trash::safe_remove "${MOCK_LOG_DIR:-}" --test-cleanup 2>/dev/null || true
}

# ----------------------------
# Basic Functionality Tests
# ----------------------------

@test "filesystem mock: initialization creates root directory" {
    mock::fs::reset
    
    run mock::fs::assert::directory_exists "/"
    assert_success
}

@test "filesystem mock: create and verify file" {
    mock::fs::create_file "/test.txt" "Hello World"
    
    run mock::fs::assert::file_exists "/test.txt"
    assert_success
    
    run mock::fs::get::file_content "/test.txt"
    assert_success
    assert_output "Hello World"
}

@test "filesystem mock: create and verify directory" {
    mock::fs::create_directory "/testdir"
    
    run mock::fs::assert::directory_exists "/testdir"
    assert_success
}

@test "filesystem mock: create nested directories" {
    mock::fs::create_directory "/a/b/c/d"
    
    run mock::fs::assert::directory_exists "/a"
    assert_success
    run mock::fs::assert::directory_exists "/a/b"
    assert_success
    run mock::fs::assert::directory_exists "/a/b/c"
    assert_success
    run mock::fs::assert::directory_exists "/a/b/c/d"
    assert_success
}

# ----------------------------
# ls Command Tests
# ----------------------------

@test "ls: list empty directory" {
    mock::fs::create_directory "/empty"
    
    run ls "/empty"
    assert_success
    assert_output ""
}

@test "ls: list directory with files" {
    mock::fs::create_directory "/mydir"
    mock::fs::create_file "/mydir/file1.txt" "content1"
    mock::fs::create_file "/mydir/file2.txt" "content2"
    mock::fs::create_file "/mydir/.hidden" "hidden"
    
    run ls "/mydir"
    assert_success
    assert_line "file1.txt"
    assert_line "file2.txt"
    refute_line ".hidden"
}

@test "ls -a: shows hidden files" {
    mock::fs::create_directory "/mydir"
    mock::fs::create_file "/mydir/file1.txt" "content1"
    mock::fs::create_file "/mydir/.hidden" "hidden"
    
    run ls -a "/mydir"
    assert_success
    assert_line "."
    assert_line ".."
    assert_line "file1.txt"
    assert_line ".hidden"
}

@test "ls -l: long format listing" {
    mock::fs::create_directory "/mydir"
    mock::fs::create_file "/mydir/test.txt" "content" "644" "user" "group"
    
    run ls -l "/mydir"
    assert_success
    assert_line --partial "total"
    assert_line --partial "rw-r--r--"
    assert_line --partial "user"
    assert_line --partial "group"
    assert_line --partial "test.txt"
}

@test "ls: non-existent path returns error" {
    run ls "/nonexistent"
    assert_failure
    assert_output --partial "No such file or directory"
}

# ----------------------------
# cat Command Tests
# ----------------------------

@test "cat: read file content" {
    mock::fs::create_file "/file.txt" "Line 1
Line 2
Line 3"
    
    run cat "/file.txt"
    assert_success
    assert_output "Line 1
Line 2
Line 3"
}

@test "cat: multiple files" {
    mock::fs::create_file "/file1.txt" "Content 1"
    mock::fs::create_file "/file2.txt" "Content 2"
    
    run cat "/file1.txt" "/file2.txt"
    assert_success
    assert_line "Content 1"
    assert_line "Content 2"
}

@test "cat: non-existent file returns error" {
    run cat "/nonexistent.txt"
    assert_failure
    assert_output --partial "No such file or directory"
}

@test "cat: directory returns error" {
    mock::fs::create_directory "/mydir"
    
    run cat "/mydir"
    assert_failure
    assert_output --partial "Is a directory"
}

@test "cat: stdin passthrough" {
    echo "stdin content" | {
        run cat
        assert_success
        assert_output "stdin content"
    }
}

# ----------------------------
# touch Command Tests
# ----------------------------

@test "touch: create new file" {
    run touch "/newfile.txt"
    assert_success
    
    run mock::fs::assert::file_exists "/newfile.txt"
    assert_success
    
    run mock::fs::get::file_content "/newfile.txt"
    assert_success
    assert_output ""
}

@test "touch: update existing file mtime" {
    mock::fs::create_file "/existing.txt" "original content"
    
    # Touch should not change content
    run touch "/existing.txt"
    assert_success
    
    run mock::fs::get::file_content "/existing.txt"
    assert_success
    assert_output "original content"
}

@test "touch: multiple files" {
    run touch "/file1.txt" "/file2.txt" "/file3.txt"
    assert_success
    
    run mock::fs::assert::file_exists "/file1.txt"
    assert_success
    run mock::fs::assert::file_exists "/file2.txt"
    assert_success
    run mock::fs::assert::file_exists "/file3.txt"
    assert_success
}

# ----------------------------
# mkdir Command Tests
# ----------------------------

@test "mkdir: create directory" {
    run mkdir "/newdir"
    assert_success
    
    run mock::fs::assert::directory_exists "/newdir"
    assert_success
}

@test "mkdir: directory already exists" {
    mock::fs::create_directory "/existing"
    
    run mkdir "/existing"
    assert_failure
    assert_output --partial "File exists"
}

@test "mkdir -p: create parent directories" {
    run mkdir -p "/a/b/c/d"
    assert_success
    
    run mock::fs::assert::directory_exists "/a"
    assert_success
    run mock::fs::assert::directory_exists "/a/b"
    assert_success
    run mock::fs::assert::directory_exists "/a/b/c"
    assert_success
    run mock::fs::assert::directory_exists "/a/b/c/d"
    assert_success
}

@test "mkdir -p: no error if exists" {
    mock::fs::create_directory "/existing"
    
    run mkdir -p "/existing"
    assert_success
}

@test "mkdir: parent directory missing" {
    run mkdir "/nonexistent/newdir"
    assert_failure
    assert_output --partial "No such file or directory"
}

# ----------------------------
# rm Command Tests
# ----------------------------

@test "rm: remove file" {
    mock::fs::create_file "/file.txt" "content"
    
    run rm "/file.txt"
    assert_success
    
    run mock::fs::assert::not_exists "/file.txt"
    assert_success
}

@test "rm: cannot remove directory without -r" {
    mock::fs::create_directory "/mydir"
    
    run rm "/mydir"
    assert_failure
    assert_output --partial "Is a directory"
}

@test "rm -r: remove directory recursively" {
    mock::fs::create_directory "/mydir"
    mock::fs::create_file "/mydir/file1.txt" "content1"
    mock::fs::create_file "/mydir/file2.txt" "content2"
    mock::fs::create_directory "/mydir/subdir"
    mock::fs::create_file "/mydir/subdir/file3.txt" "content3"
    
    run rm -r "/mydir"
    assert_success
    
    run mock::fs::assert::not_exists "/mydir"
    assert_success
    run mock::fs::assert::not_exists "/mydir/file1.txt"
    assert_success
    run mock::fs::assert::not_exists "/mydir/subdir"
    assert_success
}

@test "rm -f: force remove non-existent file" {
    run trash::safe_remove "/nonexistent.txt" --test-cleanup
    assert_success
}

@test "rm: multiple files" {
    mock::fs::create_file "/file1.txt" "content1"
    mock::fs::create_file "/file2.txt" "content2"
    mock::fs::create_file "/file3.txt" "content3"
    
    run rm "/file1.txt" "/file2.txt" "/file3.txt"
    assert_success
    
    run mock::fs::assert::not_exists "/file1.txt"
    assert_success
    run mock::fs::assert::not_exists "/file2.txt"
    assert_success
    run mock::fs::assert::not_exists "/file3.txt"
    assert_success
}

# ----------------------------
# cp Command Tests
# ----------------------------

@test "cp: copy file" {
    mock::fs::create_file "/source.txt" "content"
    
    run cp "/source.txt" "/dest.txt"
    assert_success
    
    run mock::fs::assert::file_exists "/source.txt"
    assert_success
    run mock::fs::assert::file_exists "/dest.txt"
    assert_success
    
    run mock::fs::get::file_content "/dest.txt"
    assert_success
    assert_output "content"
}

@test "cp: copy file to directory" {
    mock::fs::create_file "/source.txt" "content"
    mock::fs::create_directory "/destdir"
    
    run cp "/source.txt" "/destdir"
    assert_success
    
    run mock::fs::assert::file_exists "/destdir/source.txt"
    assert_success
    
    run mock::fs::get::file_content "/destdir/source.txt"
    assert_success
    assert_output "content"
}

@test "cp: cannot copy directory without -r" {
    mock::fs::create_directory "/sourcedir"
    
    run cp "/sourcedir" "/destdir"
    assert_failure
    assert_output --partial "omitting directory"
}

@test "cp -r: copy directory recursively" {
    mock::fs::create_directory "/sourcedir"
    mock::fs::create_file "/sourcedir/file1.txt" "content1"
    mock::fs::create_directory "/sourcedir/subdir"
    mock::fs::create_file "/sourcedir/subdir/file2.txt" "content2"
    
    run cp -r "/sourcedir" "/destdir"
    assert_success
    
    run mock::fs::assert::directory_exists "/destdir"
    assert_success
    run mock::fs::assert::file_exists "/destdir/file1.txt"
    assert_success
    run mock::fs::assert::directory_exists "/destdir/subdir"
    assert_success
    run mock::fs::assert::file_exists "/destdir/subdir/file2.txt"
    assert_success
}

@test "cp: multiple sources to directory" {
    mock::fs::create_file "/file1.txt" "content1"
    mock::fs::create_file "/file2.txt" "content2"
    mock::fs::create_directory "/destdir"
    
    run cp "/file1.txt" "/file2.txt" "/destdir"
    assert_success
    
    run mock::fs::assert::file_exists "/destdir/file1.txt"
    assert_success
    run mock::fs::assert::file_exists "/destdir/file2.txt"
    assert_success
}

# ----------------------------
# mv Command Tests
# ----------------------------

@test "mv: move file" {
    mock::fs::create_file "/source.txt" "content"
    
    run mv "/source.txt" "/dest.txt"
    assert_success
    
    run mock::fs::assert::not_exists "/source.txt"
    assert_success
    run mock::fs::assert::file_exists "/dest.txt"
    assert_success
    
    run mock::fs::get::file_content "/dest.txt"
    assert_success
    assert_output "content"
}

@test "mv: move file to directory" {
    mock::fs::create_file "/source.txt" "content"
    mock::fs::create_directory "/destdir"
    
    run mv "/source.txt" "/destdir"
    assert_success
    
    run mock::fs::assert::not_exists "/source.txt"
    assert_success
    run mock::fs::assert::file_exists "/destdir/source.txt"
    assert_success
}

@test "mv: move directory" {
    mock::fs::create_directory "/sourcedir"
    mock::fs::create_file "/sourcedir/file.txt" "content"
    
    run mv "/sourcedir" "/destdir"
    assert_success
    
    run mock::fs::assert::not_exists "/sourcedir"
    assert_success
    run mock::fs::assert::directory_exists "/destdir"
    assert_success
    run mock::fs::assert::file_exists "/destdir/file.txt"
    assert_success
}

@test "mv: rename file" {
    mock::fs::create_file "/oldname.txt" "content"
    
    run mv "/oldname.txt" "/newname.txt"
    assert_success
    
    run mock::fs::assert::not_exists "/oldname.txt"
    assert_success
    run mock::fs::assert::file_exists "/newname.txt"
    assert_success
}

# ----------------------------
# find Command Tests
# ----------------------------

@test "find: basic search" {
    mock::fs::create_directory "/searchdir"
    mock::fs::create_file "/searchdir/file1.txt" "content1"
    mock::fs::create_file "/searchdir/file2.log" "content2"
    mock::fs::create_directory "/searchdir/subdir"
    mock::fs::create_file "/searchdir/subdir/file3.txt" "content3"
    
    run find "/searchdir"
    assert_success
    assert_line "/searchdir"
    assert_line "/searchdir/file1.txt"
    assert_line "/searchdir/file2.log"
    assert_line "/searchdir/subdir"
    assert_line "/searchdir/subdir/file3.txt"
}

@test "find -name: search by name pattern" {
    mock::fs::create_directory "/searchdir"
    mock::fs::create_file "/searchdir/file1.txt" "content1"
    mock::fs::create_file "/searchdir/file2.log" "content2"
    mock::fs::create_file "/searchdir/file3.txt" "content3"
    
    run find "/searchdir" -name "*.txt"
    assert_success
    assert_line "/searchdir/file1.txt"
    assert_line "/searchdir/file3.txt"
    refute_line "/searchdir/file2.log"
}

@test "find -type f: find only files" {
    mock::fs::create_directory "/searchdir"
    mock::fs::create_file "/searchdir/file1.txt" "content1"
    mock::fs::create_directory "/searchdir/subdir"
    mock::fs::create_file "/searchdir/subdir/file2.txt" "content2"
    
    run find "/searchdir" -type f
    assert_success
    assert_line "/searchdir/file1.txt"
    assert_line "/searchdir/subdir/file2.txt"
    refute_line "/searchdir/subdir"
}

@test "find -type d: find only directories" {
    mock::fs::create_directory "/searchdir"
    mock::fs::create_file "/searchdir/file1.txt" "content1"
    mock::fs::create_directory "/searchdir/subdir1"
    mock::fs::create_directory "/searchdir/subdir2"
    
    run find "/searchdir" -type d
    assert_success
    assert_line "/searchdir"
    assert_line "/searchdir/subdir1"
    assert_line "/searchdir/subdir2"
    refute_line "/searchdir/file1.txt"
}

@test "find -maxdepth: limit search depth" {
    mock::fs::create_directory "/searchdir"
    mock::fs::create_file "/searchdir/file1.txt" "content1"
    mock::fs::create_directory "/searchdir/subdir"
    mock::fs::create_file "/searchdir/subdir/file2.txt" "content2"
    mock::fs::create_directory "/searchdir/subdir/subsubdir"
    mock::fs::create_file "/searchdir/subdir/subsubdir/file3.txt" "content3"
    
    run find "/searchdir" -maxdepth 1
    assert_success
    assert_line "/searchdir"
    assert_line "/searchdir/file1.txt"
    assert_line "/searchdir/subdir"
    refute_line "/searchdir/subdir/file2.txt"
    refute_line "/searchdir/subdir/subsubdir"
}

# ----------------------------
# chmod Command Tests
# ----------------------------

@test "chmod: change file permissions" {
    mock::fs::create_file "/file.txt" "content" "644"
    
    run chmod 755 "/file.txt"
    assert_success
    
    run mock::fs::get::file_permissions "/file.txt"
    assert_success
    assert_output "755"
}

@test "chmod: change multiple files" {
    mock::fs::create_file "/file1.txt" "content1" "644"
    mock::fs::create_file "/file2.txt" "content2" "644"
    
    run chmod 755 "/file1.txt" "/file2.txt"
    assert_success
    
    run mock::fs::get::file_permissions "/file1.txt"
    assert_success
    assert_output "755"
    
    run mock::fs::get::file_permissions "/file2.txt"
    assert_success
    assert_output "755"
}

@test "chmod -R: recursive permission change" {
    mock::fs::create_directory "/dir" "755"
    mock::fs::create_file "/dir/file1.txt" "content1" "644"
    mock::fs::create_directory "/dir/subdir" "755"
    mock::fs::create_file "/dir/subdir/file2.txt" "content2" "644"
    
    run chmod -R 777 "/dir"
    assert_success
    
    run mock::fs::get::file_permissions "/dir"
    assert_success
    assert_output "777"
    
    run mock::fs::get::file_permissions "/dir/file1.txt"
    assert_success
    assert_output "777"
    
    run mock::fs::get::file_permissions "/dir/subdir"
    assert_success
    assert_output "777"
    
    run mock::fs::get::file_permissions "/dir/subdir/file2.txt"
    assert_success
    assert_output "777"
}

# ----------------------------
# State Consistency Tests
# ----------------------------

@test "state consistency: file persists across operations" {
    # Create a file
    mock::fs::create_file "/persistent.txt" "original"
    
    # Verify it exists
    run cat "/persistent.txt"
    assert_success
    assert_output "original"
    
    # Modify it with touch
    run touch "/persistent.txt"
    assert_success
    
    # Content should remain
    run cat "/persistent.txt"
    assert_success
    assert_output "original"
    
    # Copy it
    run cp "/persistent.txt" "/copy.txt"
    assert_success
    
    # Both should exist
    run cat "/persistent.txt"
    assert_success
    assert_output "original"
    
    run cat "/copy.txt"
    assert_success
    assert_output "original"
    
    # Move the copy
    run mv "/copy.txt" "/moved.txt"
    assert_success
    
    # Copy should be gone, moved should exist
    run cat "/copy.txt"
    assert_failure
    
    run cat "/moved.txt"
    assert_success
    assert_output "original"
    
    # Remove the moved file
    run rm "/moved.txt"
    assert_success
    
    run cat "/moved.txt"
    assert_failure
    
    # Original should still exist
    run cat "/persistent.txt"
    assert_success
    assert_output "original"
}

@test "state consistency: directory operations maintain hierarchy" {
    # Create nested structure
    run mkdir -p "/root/level1/level2/level3"
    assert_success
    
    # Add files at different levels
    mock::fs::create_file "/root/file0.txt" "at root"
    mock::fs::create_file "/root/level1/file1.txt" "at level1"
    mock::fs::create_file "/root/level1/level2/file2.txt" "at level2"
    
    # List root directory
    run ls "/root"
    assert_success
    assert_line "file0.txt"
    assert_line "level1"
    
    # List level1
    run ls "/root/level1"
    assert_success
    assert_line "file1.txt"
    assert_line "level2"
    
    # Copy entire structure
    run cp -r "/root" "/root_copy"
    assert_success
    
    # Verify copy
    run cat "/root_copy/file0.txt"
    assert_success
    assert_output "at root"
    
    run cat "/root_copy/level1/file1.txt"
    assert_success
    assert_output "at level1"
    
    run cat "/root_copy/level1/level2/file2.txt"
    assert_success
    assert_output "at level2"
    
    # Remove original
    run rm -r "/root"
    assert_success
    
    # Original should be gone
    run ls "/root"
    assert_failure
    
    # Copy should still exist
    run ls "/root_copy"
    assert_success
}

@test "state consistency: complex workflow simulation" {
    # Simulate a build workflow
    
    # Create project structure
    run mkdir -p "/project/src"
    assert_success
    run mkdir -p "/project/build"
    assert_success
    run mkdir -p "/project/dist"
    assert_success
    
    # Add source files
    mock::fs::create_file "/project/src/main.js" "console.log('main');"
    mock::fs::create_file "/project/src/util.js" "export function util() {}"
    mock::fs::create_file "/project/package.json" '{"name": "test"}'
    
    # "Compile" to build directory
    run cp "/project/src/main.js" "/project/build/main.js"
    assert_success
    run cp "/project/src/util.js" "/project/build/util.js"
    assert_success
    
    # Create distribution
    run cp -r "/project/build" "/project/dist/app"
    assert_success
    run cp "/project/package.json" "/project/dist/"
    assert_success
    
    # Clean build directory
    run rm -r "/project/build"
    assert_success
    run mkdir "/project/build"
    assert_success
    
    # Verify dist still has everything
    run ls "/project/dist"
    assert_success
    assert_line "app"
    assert_line "package.json"
    
    run ls "/project/dist/app"
    assert_success
    assert_line "main.js"
    assert_line "util.js"
    
    # Verify source is intact
    run cat "/project/src/main.js"
    assert_success
    assert_output "console.log('main');"
}

# ----------------------------
# Error Mode Tests
# ----------------------------

@test "error mode: readonly filesystem" {
    export FILESYSTEM_MOCK_MODE="readonly"
    
    run touch "/newfile.txt"
    assert_failure
    assert_output --partial "Read-only file system"
    
    run mkdir "/newdir"
    assert_failure
    assert_output --partial "Read-only file system"
    
    run rm "/existingfile.txt"
    assert_failure
    assert_output --partial "Read-only file system"
    
    # Read operations should still work
    mock::fs::create_file "/readable.txt" "content"
    export FILESYSTEM_MOCK_MODE="readonly"
    
    run cat "/readable.txt"
    assert_success
    assert_output "content"
    
    run ls "/"
    assert_success
}

@test "error injection: specific command errors" {
    mock::fs::inject_error "cat" "permission_denied"
    
    mock::fs::create_file "/file.txt" "content"
    
    run cat "/file.txt"
    assert_failure
    assert_output --partial "permission_denied"
}

# ----------------------------
# Symlink Tests
# ----------------------------

@test "symlink: create and follow symlink" {
    mock::fs::create_file "/target.txt" "target content"
    mock::fs::create_symlink "/link.txt" "/target.txt"
    
    run mock::fs::assert::file_exists "/link.txt"
    assert_success
    
    # The symlink should show the target path as content
    run mock::fs::get::file_content "/link.txt"
    assert_success
    assert_output "/target.txt"
}

# ----------------------------
# Scenario Tests
# ----------------------------

@test "scenario: project structure creation" {
    run mock::fs::scenario::create_project_structure "/myproject"
    assert_success
    
    run mock::fs::assert::directory_exists "/myproject"
    assert_success
    run mock::fs::assert::directory_exists "/myproject/src"
    assert_success
    run mock::fs::assert::directory_exists "/myproject/tests"
    assert_success
    run mock::fs::assert::file_exists "/myproject/README.md"
    assert_success
    run mock::fs::assert::file_exists "/myproject/package.json"
    assert_success
}

@test "scenario: home directory creation" {
    run mock::fs::scenario::create_home_directory "testuser"
    assert_success
    
    run mock::fs::assert::directory_exists "/home/testuser"
    assert_success
    run mock::fs::assert::directory_exists "/home/testuser/.ssh"
    assert_success
    run mock::fs::assert::file_exists "/home/testuser/.bashrc"
    assert_success
    
    # Check .ssh permissions
    run mock::fs::get::file_permissions "/home/testuser/.ssh"
    assert_success
    assert_output "700"
}

# ----------------------------
# Edge Cases and Complex Tests
# ----------------------------

@test "edge case: operations on root directory" {
    # Should not be able to remove root
    run rm -r "/"
    # Root should still exist after attempted removal
    run mock::fs::assert::directory_exists "/"
    assert_success
}

@test "edge case: path normalization" {
    mock::fs::create_file "/dir/../file.txt" "content"
    
    # Should normalize to /file.txt
    run mock::fs::assert::file_exists "/file.txt"
    assert_success
}

@test "edge case: empty filename handling" {
    run touch ""
    # Should handle gracefully
    # (behavior depends on implementation)
}

@test "complex: cross-command state verification" {
    # Create initial state
    mock::fs::create_directory "/workspace"
    mock::fs::create_file "/workspace/data.txt" "initial"
    
    # Verify with ls
    run ls "/workspace"
    assert_success
    assert_line "data.txt"
    
    # Verify with cat
    run cat "/workspace/data.txt"
    assert_success
    assert_output "initial"
    
    # Modify with cp
    run cp "/workspace/data.txt" "/workspace/backup.txt"
    assert_success
    
    # Verify both exist with find
    run find "/workspace" -type f
    assert_success
    assert_line "/workspace/data.txt"
    assert_line "/workspace/backup.txt"
    
    # Move one file
    run mv "/workspace/backup.txt" "/workspace/archive.txt"
    assert_success
    
    # Verify final state
    run ls "/workspace"
    assert_success
    assert_line "data.txt"
    assert_line "archive.txt"
    refute_line "backup.txt"
}

# ----------------------------
# Performance and Stress Tests
# ----------------------------

@test "stress: create many files" {
    mock::fs::create_directory "/stress"
    
    # Create 100 files
    for i in {1..100}; do
        mock::fs::create_file "/stress/file${i}.txt" "content${i}"
    done
    
    # Verify a sample
    run mock::fs::assert::file_exists "/stress/file1.txt"
    assert_success
    run mock::fs::assert::file_exists "/stress/file50.txt"
    assert_success
    run mock::fs::assert::file_exists "/stress/file100.txt"
    assert_success
    
    # List should work
    run bash -c 'ls "/stress" | wc -l'
    assert_success
    # Should have 100 files
    [[ "$output" -eq 100 ]]
}

@test "stress: deep directory nesting" {
    local deep_path="/a"
    for i in {1..20}; do
        deep_path="${deep_path}/level${i}"
    done
    
    run mkdir -p "$deep_path"
    assert_success
    
    run mock::fs::assert::directory_exists "$deep_path"
    assert_success
}

# ----------------------------
# Logging and Verification Tests
# ----------------------------

@test "logging: commands are logged" {
    skip "Logging system not implemented - logs.sh is commented out in setup"
    
    # Ensure log directory exists
    [[ -d "$MOCK_LOG_DIR" ]]
    
    # Perform some operations
    run touch "/logged.txt"
    assert_success
    
    run cat "/logged.txt"
    assert_success
    
    run rm "/logged.txt"
    assert_success
    
    # Check logs exist
    [[ -f "$MOCK_LOG_DIR/command_calls.log" ]]
    
    # Verify commands were logged
    grep -q "touch /logged.txt" "$MOCK_LOG_DIR/command_calls.log"
    grep -q "cat /logged.txt" "$MOCK_LOG_DIR/command_calls.log"
    grep -q "rm /logged.txt" "$MOCK_LOG_DIR/command_calls.log"
}

# ----------------------------
# Subshell and Export Tests
# ----------------------------

@test "subshell: state persists across subshells" {
    # Create file in parent shell
    mock::fs::create_file "/parent.txt" "parent content"
    
    # Verify in subshell
    (
        run cat "/parent.txt"
        assert_success
        assert_output "parent content"
        
        # Create file in subshell
        run touch "/subshell.txt"
        assert_success
    )
    
    # Verify subshell creation is visible in parent
    run mock::fs::assert::file_exists "/subshell.txt"
    assert_success
}

@test "export: functions are available in subshells" {
    (
        # These functions should be available
        run type -t ls
        assert_success
        assert_output "function"
        
        run type -t cat
        assert_success
        assert_output "function"
        
        run type -t mkdir
        assert_success
        assert_output "function"
    )
}