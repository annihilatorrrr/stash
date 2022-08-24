import requests

class StashInterface:
	port = ""
	url = ""
	headers = {
		"Accept-Encoding": "gzip, deflate, br",
		"Content-Type": "application/json",
		"Accept": "application/json",
		"Connection": "keep-alive",
		"DNT": "1"
		}

	def __init__(self, conn):
		self.port = conn['Port']
		scheme = conn['Scheme']

		self.url = f"{scheme}://localhost:{str(self.port)}/graphql"

		# Session cookie for authentication
		self.cookies = {
			'session': conn.get('SessionCookie').get('Value')
		}

	def __callGraphQL(self, query, variables = None):
		json = {'query': query}
		if variables != None:
			json['variables'] = variables

		# handle cookies
		response = requests.post(self.url, json=json, headers=self.headers, cookies=self.cookies)

		if response.status_code != 200:
			raise Exception(
				f"GraphQL query failed:{response.status_code} - {response.content}. Query: {query}. Variables: {variables}"
			)

		result = response.json()
		if result.get("error", None):
			for error in result["error"]["errors"]:
				raise Exception(f"GraphQL error: {error}")
		if result.get("data", None):
			return result.get("data")

	def findTagIdWithName(self, name):
		query = """
query {
  allTags {
    id
    name
  }
}
		"""

		result = self.__callGraphQL(query)

		return next(
			(tag["id"] for tag in result["allTags"] if tag["name"] == name), None
		)

	def createTagWithName(self, name):
		query = """
mutation tagCreate($input:TagCreateInput!) {
  tagCreate(input: $input){
    id       
  }
}
"""
		variables = {'input': {
			'name': name
		}}

		result = self.__callGraphQL(query, variables)
		return result["tagCreate"]["id"]

	def destroyTag(self, id):
		query = """
mutation tagDestroy($input: TagDestroyInput!) {
  tagDestroy(input: $input)
}
"""
		variables = {'input': {
			'id': id
		}}

		self.__callGraphQL(query, variables)

	def findRandomSceneId(self):
		query = """
query findScenes($filter: FindFilterType!) {
  findScenes(filter: $filter) {
    count
    scenes {
      id
      tags {
        id
      }
    }
  }
}
"""

		variables = {'filter': {
			'per_page': 1,
			'sort': 'random'
		}}

		result = self.__callGraphQL(query, variables)

		if result["findScenes"]["count"] == 0:
			return None

		return result["findScenes"]["scenes"][0]

	def updateScene(self, sceneData):
		query = """
mutation sceneUpdate($input:SceneUpdateInput!) {
  sceneUpdate(input: $input) {
    id
  }
}
"""
		variables = {'input': sceneData}

		self.__callGraphQL(query, variables)