#!/usr/bin/env bash
#  vim:ts=4:sts=4:sw=4:et
#
#  Author: Hari Sekhon
#  Date: 2024-10-02 06:46:42 +0300 (Wed, 02 Oct 2024)
#
#  https///github.com/HariSekhon/GitHub-Repos-MermaidJS-Gantt-Chart
#
#  License: see accompanying Hari Sekhon LICENSE file
#
#  If you're using my code you're welcome to connect with me on LinkedIn and optionally send me feedback to help steer this or other code I publish
#
#  https://www.linkedin.com/in/HariSekhon
#

set -euo pipefail
[ -n "${DEBUG:-}" ] && set -x
srcdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1090,SC1091
. "$srcdir/bash-tools/lib/utils.sh"

# shellcheck disable=SC2034,SC2154
usage_description="
Replaces the Gantt Chart blocks in the README.md file
"

# used by usage() in lib/utils.sh
# shellcheck disable=SC2034
usage_args=""

help_usage "$@"

num_args 0 "$@"

cd "$srcdir"

timestamp "Starting main.go to generate the gantt_chart.md file"
echo

go run main.go HariSekhon

echo

if uname | grep -q Darwin; then
    sed(){
        command gsed "$@"
    }
fi

markdown_file="README.md"
markdown_tmp="$(mktemp)"

if ! [ -f "$markdown_file" ]; then
    die "File not found: $markdown_file"
fi

# check the tags existing in the markdown file otherwise we can't do anything
for x in GANTT_CHART_START GANTT_CHART_END GANTT_CHART2_START GANTT_CHART2_END; do
    if ! grep -q "<!--.*$x.*-->" "$markdown_file"; then
        die "Markdown file '$markdown_file' is missing the index boundary comment <!--.*$x.*-->"
    fi
done

timestamp "Replacing index in file: $markdown_file"

sed -n "
    1,/GANTT_CHART_START/p

    /GANTT_CHART_START/ a

    /GANTT_CHART_START/,/GANTT_CHART_END/ {
        /GANTT_CHART_START/ {
            a \`\`\`none
            r gantt_chart.mmd
            a \`\`\`
        }
    }

    /GANTT_CHART_END/ i

    /GANTT_CHART_END/,$ p
" "$markdown_file" > "$markdown_tmp"

mv -f "$markdown_tmp" "$markdown_file"

sed -n "
    1,/GANTT_CHART2_START/p

    /GANTT_CHART2_START/ a

    /GANTT_CHART2_START/,/GANTT_CHART2_END/ {
        /GANTT_CHART2_START/ {
            a \`\`\`mermaid
            r gantt_chart.mmd
            a \`\`\`
        }
    }

    /GANTT_CHART2_END/ i

    /GANTT_CHART2_END/,$ p
" "$markdown_file" > "$markdown_tmp"

mv -f "$markdown_tmp" "$markdown_file"
