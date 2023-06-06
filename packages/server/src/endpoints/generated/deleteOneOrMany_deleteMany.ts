export const deleteOneOrMany_deleteMany = {
  "fieldName": "deleteMany",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "deleteMany",
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
              "value": "count",
              "loc": {
                "start": 83,
                "end": 88
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 88
            }
          }
        ],
        "loc": {
          "start": 77,
          "end": 92
        }
      },
      "loc": {
        "start": 51,
        "end": 92
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
      "value": "deleteMany",
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
              "value": "DeleteManyInput",
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
            "value": "deleteMany",
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
                  "value": "count",
                  "loc": {
                    "start": 83,
                    "end": 88
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 83,
                  "end": 88
                }
              }
            ],
            "loc": {
              "start": 77,
              "end": 92
            }
          },
          "loc": {
            "start": 51,
            "end": 92
          }
        }
      ],
      "loc": {
        "start": 47,
        "end": 94
      }
    },
    "loc": {
      "start": 1,
      "end": 94
    }
  },
  "variableValues": {},
  "path": {
    "key": "deleteOneOrMany_deleteMany"
  }
} as const;
