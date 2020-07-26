#!/bin/bash
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export PROMPT_TIMEOUT=3

########################
# include the magic
########################
. demo-magic.sh

rm -fr output

# hide the evidence
clear

pwd

bold=$(tput bold)
normal=$(tput sgr0)

# start demo
clear
p "# we have two kpt packages"
pe "ls -d */"
p ""
wait


p "# package foo contains two resources and includes setters and substitutions"
pe "cat foo/foo.yaml"
pe "cat foo/bar.yaml"
pe "kpt cfg list-setters foo --include-subst"
p ""
wait

p "# package my-pkg contains two LocalPackage resources that both reference the foo package but with different setter values"
pe "kpt cfg tree my-pkg"
pe "cat my-pkg/minnesota-pkg.yaml"
pe "cat my-pkg/washington-pkg.yaml"
p ""
wait

p "# running the localpkg-fn kpt function replaces the LocalPackage resources with the output of package foo after using the setters"
pe "mkdir output"
pe "kpt fn source my-pkg | kpt fn run --enable-exec --exec-path ./localpkg-fn | kpt fn sink output"
p ""
wait

p "# the output has two version of the Foo resource and two versions of the Bar resource"
pe "kpt cfg tree output"
pe "cat output/foo.yaml"
pe "cat output/bar.yaml"



