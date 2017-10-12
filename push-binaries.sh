#!/bin/sh
# Push binaries to quasar-binaries repo
make binaries
echo The gh_token is $GH_TOKEN
cd binaries && git remote add binaries https://$GH_TOKEN@github.com/charly3pins/quasar-binaries
git push binaries master
if [ $? -ne 0 ]; then
  cd .. && rm -rf binaries
  exit 1
fi
