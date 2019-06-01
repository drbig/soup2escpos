#!/bin/bash
set -e

EXE=$1
OUTDIR=$(mktemp -d /tmp/soup2esc-XXXXXX)

for test_path in ./tests/inputs/*.html; do
  testfile=$(basename $test_path)
  snapfile="./tests/snapshots/$testfile"
  outfile="$OUTDIR/$testfile-out"
  echo "TEST $testfile"
  $EXE $test_path > $outfile
  diff $outfile $snapfile
done

rm -rf $OUTDIR
