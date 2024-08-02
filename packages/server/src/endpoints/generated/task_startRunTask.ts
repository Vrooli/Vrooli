export const task_startRunTask = {
  "fieldName": "startRunTask",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startRunTask",
        "loc": {
          "start": 55,
          "end": 67
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 68,
              "end": 73
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 76,
                "end": 81
              }
            },
            "loc": {
              "start": 75,
              "end": 81
            }
          },
          "loc": {
            "start": 68,
            "end": 81
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "success",
              "loc": {
                "start": 89,
                "end": 96
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 89,
              "end": 96
            }
          }
        ],
        "loc": {
          "start": 83,
          "end": 100
        }
      },
      "loc": {
        "start": 55,
        "end": 100
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {},
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "startRunTask",
      "loc": {
        "start": 10,
        "end": 22
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 24,
              "end": 29
            }
          },
          "loc": {
            "start": 23,
            "end": 29
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "StartRunTaskInput",
              "loc": {
                "start": 31,
                "end": 48
              }
            },
            "loc": {
              "start": 31,
              "end": 48
            }
          },
          "loc": {
            "start": 31,
            "end": 49
          }
        },
        "directives": [],
        "loc": {
          "start": 23,
          "end": 49
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "startRunTask",
            "loc": {
              "start": 55,
              "end": 67
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 68,
                  "end": 73
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 76,
                    "end": 81
                  }
                },
                "loc": {
                  "start": 75,
                  "end": 81
                }
              },
              "loc": {
                "start": 68,
                "end": 81
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "success",
                  "loc": {
                    "start": 89,
                    "end": 96
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 89,
                  "end": 96
                }
              }
            ],
            "loc": {
              "start": 83,
              "end": 100
            }
          },
          "loc": {
            "start": 55,
            "end": 100
          }
        }
      ],
      "loc": {
        "start": 51,
        "end": 102
      }
    },
    "loc": {
      "start": 1,
      "end": 102
    }
  },
  "variableValues": {},
  "path": {
    "key": "task_startRunTask"
  }
} as const;
