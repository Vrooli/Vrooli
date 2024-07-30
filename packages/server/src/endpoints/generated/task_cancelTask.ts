export const task_cancelTask = {
  "fieldName": "cancelTask",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "cancelTask",
        "loc": {
          "start": 51,
          "end": 61
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 62,
              "end": 67
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 70,
                "end": 75
              }
            },
            "loc": {
              "start": 69,
              "end": 75
            }
          },
          "loc": {
            "start": 62,
            "end": 75
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
                "start": 83,
                "end": 90
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 90
            }
          }
        ],
        "loc": {
          "start": 77,
          "end": 94
        }
      },
      "loc": {
        "start": 51,
        "end": 94
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
      "value": "cancelTask",
      "loc": {
        "start": 10,
        "end": 20
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
              "start": 22,
              "end": 27
            }
          },
          "loc": {
            "start": 21,
            "end": 27
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "CancelTaskInput",
              "loc": {
                "start": 29,
                "end": 44
              }
            },
            "loc": {
              "start": 29,
              "end": 44
            }
          },
          "loc": {
            "start": 29,
            "end": 45
          }
        },
        "directives": [],
        "loc": {
          "start": 21,
          "end": 45
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
            "value": "cancelTask",
            "loc": {
              "start": 51,
              "end": 61
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 62,
                  "end": 67
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 70,
                    "end": 75
                  }
                },
                "loc": {
                  "start": 69,
                  "end": 75
                }
              },
              "loc": {
                "start": 62,
                "end": 75
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
                    "start": 83,
                    "end": 90
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 83,
                  "end": 90
                }
              }
            ],
            "loc": {
              "start": 77,
              "end": 94
            }
          },
          "loc": {
            "start": 51,
            "end": 94
          }
        }
      ],
      "loc": {
        "start": 47,
        "end": 96
      }
    },
    "loc": {
      "start": 1,
      "end": 96
    }
  },
  "variableValues": {},
  "path": {
    "key": "task_cancelTask"
  }
} as const;
