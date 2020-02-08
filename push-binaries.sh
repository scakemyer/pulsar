#!/bin/sh
# Push binaries to magnetar-binaries repo
make binaries
cd binaries && git remote add binaries https://$GH_TOKEN@github.com/charly3pins/magnetar-binaries
git push binaries master
if [ $? -ne 0 ]; then
  cd .. && rm -rf binaries
  exit 1
fi
