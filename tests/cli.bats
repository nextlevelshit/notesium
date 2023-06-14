#!/usr/bin/env bats

setup_file() {
    [ -e "/tmp/notesium-test-corpus" ] && exit 1
    run mkdir /tmp/notesium-test-corpus
    export NOTESIUM_DIR="/tmp/notesium-test-corpus"
    export PATH="$(realpath $BATS_TEST_DIRNAME/../):$PATH"
}

teardown_file() {
    run rmdir /tmp/notesium-test-corpus
}

@test "cli: print usage if no arguments specified" {
    skip
    run notesium
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == 'Usage: notesium COMMAND [OPTIONS]' ]
}

@test "cli: print usage if -h --help help" {
    skip
    run notesium -h
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == 'Usage: notesium COMMAND [OPTIONS]' ]

    run notesium --help
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == 'Usage: notesium COMMAND [OPTIONS]' ]

    run notesium help
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == 'Usage: notesium COMMAND [OPTIONS]' ]
}

@test "cli: version command sniff test" {
    skip
    run notesium -v
    echo "$output"
    [ $status -eq 0 ]

    run notesium --version
    echo "$output"
    [ $status -eq 0 ]

    run notesium version
    echo "$output"
    [ $status -eq 0 ]
}

@test "cli: non-existent command fatal error" {
    skip
    run notesium non-existent
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == 'Fatal: unrecognized command: non-existent' ]
}

@test "cli: non-existent option fatal error" {
    skip
    run notesium --non-existent
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == 'Fatal: unrecognized option: --non-existent' ]
}

@test "cli: home error if NOTESIUM_DIR does not exist" {
    skip
    export NOTESIUM_DIR="/tmp/notesium-test-foo"
    run notesium home
    echo "$output"
    [ $status -eq 1 ]
    [ "${lines[0]}" == "Fatal: NOTESIUM_DIR does not exist: $NOTESIUM_DIR" ]
}

@test "cli: home prints default NOTESIUM_DIR if not set" {
    skip
    [ -e "$HOME/notes" ] || skip "$HOME/notes does not exist"
    unset NOTESIUM_DIR
    run notesium home
    echo "$output"
    [ $status -eq 0 ]
    [ "${lines[0]}" == "$(realpath $HOME/notes)" ]
}

@test "cli: home prints NOTESIUM_DIR upon successful verification" {
    skip
    run notesium home
    echo "$output"
    [ $status -eq 0 ]
    [ "${lines[0]}" == "/tmp/notesium-test-corpus" ]
}

@test "cli: new dirname equal to NOTESIUM_DIR realpath" {
    skip
    run notesium new
    echo "$output"
    [ $status -eq 0 ]
    [ "$(dirname $output)" == "/tmp/notesium-test-corpus" ]
}

@test "cli: new basename is 8 chars plus .md extension" {
    skip
    run notesium new
    echo "$output"
    [ $status -eq 0 ]
    [ "$(basename $output | tr -d '\n' | wc -c)" == "11" ]
    [ "$(basename --suffix=.md $output | tr -d '\n' | wc -c)" == "8" ]
}

@test "cli: new basename is hex for now epoch (within 10s range)" {
    skip
    run notesium new
    echo "$output"
    [ $status -eq 0 ]
    epoch="$(printf '%d' 0x$(basename --suffix=.md $output))"
    [ "$epoch" -gt "$(date -d "-5 seconds" +%s)" ]
    [ "$epoch" -lt "$(date -d "+5 seconds" +%s)" ]
}
