export const deleteOneOrMany_deleteOne = {
  "fieldName": "deleteOne",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "deleteOne",
        "loc": {
          "start": 49,
          "end": 58
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 59,
              "end": 64
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 67,
                "end": 72
              }
            },
            "loc": {
              "start": 66,
              "end": 72
            }
          },
          "loc": {
            "start": 59,
            "end": 72
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
                "start": 80,
                "end": 87
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 80,
              "end": 87
            }
          }
        ],
        "loc": {
          "start": 74,
          "end": 91
        }
      },
      "loc": {
        "start": 49,
        "end": 91
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
      "value": "deleteOne",
      "loc": {
        "start": 10,
        "end": 19
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
              "start": 21,
              "end": 26
            }
          },
          "loc": {
            "start": 20,
            "end": 26
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "DeleteOneInput",
              "loc": {
                "start": 28,
                "end": 42
              }
            },
            "loc": {
              "start": 28,
              "end": 42
            }
          },
          "loc": {
            "start": 28,
            "end": 43
          }
        },
        "directives": [],
        "loc": {
          "start": 20,
          "end": 43
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
            "value": "deleteOne",
            "loc": {
              "start": 49,
              "end": 58
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 59,
                  "end": 64
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 67,
                    "end": 72
                  }
                },
                "loc": {
                  "start": 66,
                  "end": 72
                }
              },
              "loc": {
                "start": 59,
                "end": 72
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
                    "start": 80,
                    "end": 87
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 80,
                  "end": 87
                }
              }
            ],
            "loc": {
              "start": 74,
              "end": 91
            }
          },
          "loc": {
            "start": 49,
            "end": 91
          }
        }
      ],
      "loc": {
        "start": 45,
        "end": 93
      }
    },
    "loc": {
      "start": 1,
      "end": 93
    }
  },
  "variableValues": {},
  "path": {
    "key": "deleteOneOrMany_deleteOne"
  }
};
