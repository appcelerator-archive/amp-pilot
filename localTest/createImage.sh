echo amp-pilot should be built first in the upper directory (go build)
cp ../amp-pilot .
docker build -t amp-test .
