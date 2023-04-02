#!/bin/sh
set -e

fatal() { echo "Fatal: $*" 1>&2; exit 1; }

usage() {
cat<<EOF
Usage: $(basename "$0") COMMAND [OPTIONS]

Commands:
  new               Print path for a new note
  home              Print path to notes directory
  list              Print list of notes
    --color         Color code prefix using ansi escape sequences
    --labels        Limit list to only label notes (ie. one word title)
    --orphans       Limit list to notes without forward or back links
    --match=PATTERN Limit list to notes where pattern appears
    --sort=WORD     Sort list by title or modification time (mtime|title)
    --prefix=WORD   Include linked labels or modification date (mtime|label)
  links [filename]  Print list of links
    --color         Color code using ansi escape sequences
    --outgoing      Limit list to outgoing links related to filename
    --incoming      Limit list to incoming links related to filename
    --dangling      Limit list to broken links
  lines             Print all lines of notes (ie. fulltext search)
    --color         Color code prefix using ansi escape sequences
    --prefix=title  Include note title as prefix of each line

Environment:
  NOTESIUM_DIR      Path to notes directory (default: \$HOME/notes)

EOF
exit 1
}


_list() {
    awk 'FNR==1{print FILENAME ":1:", substr($0,3)}' $@
}
_list_prefix_mtime() {
    ls -l $1 --time-style=long-iso *.md | \
        awk -v fname_col=8 -v C=$Color -v R=$Reset '
            {fname=$fname_col; getline firstline < fname;
            print fname ":1:", C $6 R, substr(firstline,3)}'
}
_list_prefix_label_awk() {
    awk -F ":" -v fname_col=2 '
        {fname=$fname_col; getline firstline < fname;
        print $1 ":" substr(firstline,3); close(fname)}' | \
        awk -F ":" -v fname_col=1 -v C=$Color -v R=$Reset '
            {fname=$fname_col; getline firstline < fname;
            print fname ":1:", C $2 R, substr(firstline,3); close(fname)}'
}
_list_prefix_label_sort() {
    labels="$(awk 'FNR==1 && NF==2 {printf "-e %s ", FILENAME}' *.md)"
    grep --only-matching $labels $@ | _list_prefix_label_awk | sort -k2
    _list $(grep --files-without-match $labels $@) | sort -k2
}

_list_prefix_label() {
    labels="$(awk 'FNR==1 && NF==2 {printf "-e %s ", FILENAME}' *.md)"
    grep --only-matching $labels $@ | _list_prefix_label_awk
    _list $(grep --files-without-match $labels $@)
}
_list_match() {
    pattern="$1"; shift
    [ "$pattern" ] || fatal "pattern not specified"
    grep --line-number --only-matching $pattern $@ | \
        awk -F ":" -v fname_col=1 '
            {fname=$fname_col; getline firstline < fname;
            print $1 ":" $2 ":", substr(firstline,3); close(fname)}' | uniq
}
_list_labels() {
    awk 'FNR==1 && NF==2 {print FILENAME ":1:", substr($0,3)}' $@
}
_list_orphans() {
    re='[[:alnum:]]\{8\}\.md'
    existing_links="$(grep --no-filename --only-match $re *.md | \
                      awk '{printf "-e %s ", $0}')"
    orphans=$(grep --files-without-match $re $@ | grep -v $existing_links)
    [ "$orphans" ] || return 0
    _list $orphans
}
_links() {
    re='[[:alnum:]]\{8\}\.md'
    grep --with-filename --line-number --only-match $re $@ | \
        awk -F ":" -v fname_col=1 '
            {fname=$fname_col; getline firstline < fname;
            print $1 ":" $2 ":" ";;" substr(firstline,3) ";;" $3;
            close(fname)}' | \
            awk -F ";;" -v fname_col=3 -v C=$Color -v R=$Reset '
                {fname=$fname_col; getline firstline < fname;
                print $1, C $2 R, "→", substr(firstline,3);
                close(fname)}'
}
_links_dangling() {
    re='[[:alnum:]]\{8\}\.md'
    notelist="$(ls *.md | awk '{printf "-e %s ", $0}')"
    dangling="$(grep --no-filename --only-match $re *.md | sort | uniq | \
                grep -v $notelist | awk '{printf "-e %s ", $0}')"
    [ "$dangling" ] || return 0
    grep --with-filename --line-number --only-matching $dangling $@ | \
        awk -F ":" -v fname_col=1 -v C=$Color -v R=$Reset '
            {fname=$fname_col; getline firstline < fname;
            print fname ":" $2 ":", C substr(firstline,3) R, "→", $3;
            close(fname)}'
}
_links_outgoing() {
    [ "$1" ] || fatal "filename not specified"
    outgoing="$(grep --only-matching '\([[:alnum:]]\{8\}\.md\)' $1)"
    [ "$outgoing" ] || return 0
    _list $outgoing
}
_links_incoming() {
    [ "$1" ] || fatal "filename not specified"
    _list_match ]\($1\) *.md
}
_links_outgoing_prefix() {
    _links_outgoing $1 | \
        awk -v C=$Color -v R=$Reset '{$1=$1 " " C "outgoing" R}1'
}
_links_incoming_prefix() {
    _links_incoming $1 | \
        awk -v C=$Color -v R=$Reset '{$1=$1 " " C "incoming" R}1'
}
_lines() {
    awk 'NF {print FILENAME ":" FNR ": " $0}' $@
}
_lines_prefix_title() {
    awk -v C=$Color -v R=$Reset '
        FNR == 1 { title=substr($0,3) }
        NF {print FILENAME ":" FNR ":", C title R, $0}' $@
}


notesium_list() {
    while [ "$1" != "" ]; do
        case $1 in
            --color)                Color="\033[0;36m"; Reset="\033[0m";;
            --sort=title)           Sort="SortTitle";;
            --sort=mtime)           Sort="SortMtime";;
            --prefix=label)         Prefix="PrefixLabel";;
            --prefix=mtime)         Prefix="PrefixMtime";;
            --labels)               Limit="LimitLabels";;
            --orphans)              Limit="LimitOrphans";;
            --match=*)              Limit="LimitMatch"; pattern="${1##*=}";;
            *)                      fatal "unrecognized option: $1";;
        esac
        shift
    done
    case List${Limit}${Prefix}${Sort} in
        List)                       _list *.md;;
        ListSortTitle)              _list *.md | sort -k2;;
        ListSortMtime)              _list $(ls -t *.md);;
        ListPrefixLabel)            _list_prefix_label *.md;;
        ListPrefixLabelSortTitle)   _list_prefix_label_sort *.md;;
        ListPrefixLabelSortMtime)   _list_prefix_label $(ls -t *.md);;
        ListPrefixMtime)            _list_prefix_mtime;;
        ListPrefixMtimeSortTitle)   _list_prefix_mtime | sort -k3;;
        ListPrefixMtimeSortMtime)   _list_prefix_mtime -t;;
        ListLimitLabels)            _list_labels *.md;;
        ListLimitLabelsSortTitle)   _list_labels *.md | sort -k2;;
        ListLimitLabelsSortMtime)   _list_labels $(ls -t *.md);;
        ListLimitOrphans)           _list_orphans *.md;;
        ListLimitOrphansSortTitle)  _list_orphans *.md | sort -k2;;
        ListLimitOrphansSortMtime)  _list_orphans $(ls -t *.md);;
        ListLimitMatch)             _list_match $pattern *.md;;
        ListLimitMatchSortTitle)    _list_match $pattern *.md | sort -k2;;
        ListLimitMatchSortMtime)    _list_match $pattern $(ls -t *.md);;
        *)                          fatal "unsupported option grouping";;
    esac
}

notesium_links() {
    unset fname
    _set_fname() {
        [ "$fname" ] && fatal "unrecognized option: $1"
        [ -e "$1" ] || fatal "does not exist: $1"
        fname=$1
    }
    while [ "$1" != "" ]; do
        case $1 in
            --color)                Color="\033[0;36m"; Reset="\033[0m";;
            --outgoing)             Outgoing="Outgoing";;
            --incoming)             Incoming="Incoming";;
            --dangling)             Dangling="Dangling";;
            *)                      _set_fname $1;
        esac
        shift
    done
    if [ "$fname" ]; then
        case ${Dangling}${Outgoing}${Incoming} in
            Dangling*)              fatal "dangling filename not supported";;
            "")                     Outgoing="Outgoing"; Incoming="Incoming";;
        esac
    fi
    case Links${Dangling}${Outgoing}${Incoming} in
        Links)                      _links *.md | sort -k2;;
        LinksDangling)              _links_dangling *.md | sort -k2;;
        LinksOutgoing)              _links_outgoing $fname;;
        LinksIncoming)              _links_incoming $fname | sort -k2;;
        LinksOutgoingIncoming)      _links_outgoing_prefix $fname;
                                    _links_incoming_prefix $fname | sort -k3;;
        *)                          fatal "unsupported option grouping";;
    esac
}

notesium_lines() {
    while [ "$1" != "" ]; do
        case $1 in
            --color)                Color="\033[0;36m"; Reset="\033[0m";;
            --prefix=title)         Prefix="PrefixTitle";;
            *)                      fatal "unrecognized option: $1";;
        esac
        shift
    done
    case Lines${Prefix} in
        Lines)                      _lines *.md;;
        LinesPrefixTitle)           _lines_prefix_title *.md;;
        *)                          fatal "unsupported option grouping";;
    esac
}

main() {
    case $1 in ""|-h|--help|help) usage;; esac

    NOTESIUM_DIR="${NOTESIUM_DIR:=$HOME/notes}"
    NOTESIUM_DIR="$(realpath $NOTESIUM_DIR)"
    [ -d "$NOTESIUM_DIR" ] || \
        fatal "NOTESIUM_DIR does not exist: $NOTESIUM_DIR"
    cd $NOTESIUM_DIR

    case $1 in
        new)        echo "$NOTESIUM_DIR/$(mcookie | head -c8).md";;
        home)       echo "$NOTESIUM_DIR";;
        list)       shift; notesium_list $@;;
        links)      shift; notesium_links $@;;
        lines)      shift; notesium_lines $@;;
        -*)         fatal "unrecognized option: $1";;
        *)          fatal "unrecognized command: $1";;
    esac
}

main "$@"
