#!/usr/bin/env bats

setup_file() {
    export NOTESIUM_DIR="$BATS_TEST_DIRNAME/fixtures"
    export PATH="$(realpath $BATS_TEST_DIRNAME/../):$PATH"
}

@test "lines: default" {
    run notesium.sh lines
    echo "$output"
    [ $status -eq 0 ]
    [ "${lines[0]}" == "0c8bea98.md:1: # book" ]
    [ "${lines[1]}" == "26f89821.md:1: # lorem ipsum" ]
    [ "${lines[3]}" == "26f89821.md:4: tempor incididunt ut labore et dolore magna aliqua." ]
}

@test "lines: prefix title" {
    run notesium.sh lines --prefix=title
    echo "$output"
    [ $status -eq 0 ]
    [ "${lines[0]}" == "0c8bea98.md:1: book # book" ]
    [ "${lines[1]}" == "26f89821.md:1: lorem ipsum # lorem ipsum" ]
}