import json
import sys
import time

import log
from stash_interface import StashInterface

# raw plugins may accept the plugin input from stdin, or they can elect
# to ignore it entirely. In this case it optionally reads from the
# command-line parameters.
def main():
	input = None

	if len(sys.argv) < 2:
		input = readJSONInput()
		log.LogDebug(f"Raw input: {json.dumps(input)}")
	else:
		log.LogDebug("Using command line inputs")
		mode = sys.argv[1]
		log.LogDebug(f"Command line inputs: {sys.argv[1:]}")

		input = {
			'args': {"mode": mode},
			'server_connection': {"Scheme": "http", "Port": 9999},
		}

	output = {}
	run(input, output)

	out = json.dumps(output)
	print(out + "\n")

def readJSONInput():
	input = sys.stdin.read()
	return json.loads(input)

def run(input, output):
	modeArg = input['args']["mode"]

	try:
		if modeArg in ["", "add"]:
			client = StashInterface(input["server_connection"])
			addTag(client)
		elif modeArg == "remove":
			client = StashInterface(input["server_connection"])
			removeTag(client)
		elif modeArg == "long":
			doLongTask()
		elif modeArg == "indef":
			doIndefiniteTask()
	except Exception as e:
		raise
		#output["error"] = str(e)
		#return

	output["output"] = "ok"

def doLongTask():
	total = 100
	log.LogInfo("Doing long task")
	for upTo in range(total):
		time.sleep(1)

		log.LogProgress(float(upTo) / float(total))

def doIndefiniteTask():
	log.LogWarning("Sleeping indefinitely")
	while True:
		time.sleep(1)

def addTag(client):
	tagName = "Hawwwwt"
	tagID = client.findTagIdWithName(tagName)

	if tagID is None:
		tagID = client.createTagWithName(tagName)

	scene = client.findRandomSceneId()

	if scene is None:
		raise Exception("no scenes to add tag to")

	tagIds = [t["id"] for t in scene["tags"]]
	# remove first to ensure we don't re-add the same id
	try:
		tagIds.remove(tagID)
	except ValueError:
		pass

	tagIds.append(tagID)

	input = {
		"id": scene["id"],
		"tag_ids": tagIds
	}

	log.LogInfo(f'Adding tag to scene {scene["id"]}')
	client.updateScene(input)

def removeTag(client):
	tagName = "Hawwwwt"
	tagID = client.findTagIdWithName(tagName)

	if tagID is None:
		log.LogInfo("Tag does not exist. Nothing to remove")
		return

	log.LogInfo("Destroying tag")
	client.destroyTag(tagID)

main()