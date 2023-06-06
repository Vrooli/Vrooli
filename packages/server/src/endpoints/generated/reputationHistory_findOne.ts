export const reputationHistory_findOne = {
  "fieldName": "reputationHistory",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reputationHistory",
        "loc": {
          "start": 53,
          "end": 70
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 71,
              "end": 76
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 79,
                "end": 84
              }
            },
            "loc": {
              "start": 78,
              "end": 84
            }
          },
          "loc": {
            "start": 71,
            "end": 84
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
              "value": "id",
              "loc": {
                "start": 92,
                "end": 94
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 92,
              "end": 94
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 99,
                "end": 109
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 99,
              "end": 109
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 114,
                "end": 124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 114,
              "end": 124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "amount",
              "loc": {
                "start": 129,
                "end": 135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 129,
              "end": 135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "event",
              "loc": {
                "start": 140,
                "end": 145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 140,
              "end": 145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "objectId1",
              "loc": {
                "start": 150,
                "end": 159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 150,
              "end": 159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "objectId2",
              "loc": {
                "start": 164,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 164,
              "end": 173
            }
          }
        ],
        "loc": {
          "start": 86,
          "end": 177
        }
      },
      "loc": {
        "start": 53,
        "end": 177
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
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "reputationHistory",
      "loc": {
        "start": 7,
        "end": 24
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
              "start": 26,
              "end": 31
            }
          },
          "loc": {
            "start": 25,
            "end": 31
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "FindByIdInput",
              "loc": {
                "start": 33,
                "end": 46
              }
            },
            "loc": {
              "start": 33,
              "end": 46
            }
          },
          "loc": {
            "start": 33,
            "end": 47
          }
        },
        "directives": [],
        "loc": {
          "start": 25,
          "end": 47
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
            "value": "reputationHistory",
            "loc": {
              "start": 53,
              "end": 70
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 71,
                  "end": 76
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 79,
                    "end": 84
                  }
                },
                "loc": {
                  "start": 78,
                  "end": 84
                }
              },
              "loc": {
                "start": 71,
                "end": 84
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
                  "value": "id",
                  "loc": {
                    "start": 92,
                    "end": 94
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 92,
                  "end": 94
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "created_at",
                  "loc": {
                    "start": 99,
                    "end": 109
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 99,
                  "end": 109
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "updated_at",
                  "loc": {
                    "start": 114,
                    "end": 124
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 114,
                  "end": 124
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "amount",
                  "loc": {
                    "start": 129,
                    "end": 135
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 129,
                  "end": 135
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "event",
                  "loc": {
                    "start": 140,
                    "end": 145
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 140,
                  "end": 145
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "objectId1",
                  "loc": {
                    "start": 150,
                    "end": 159
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 150,
                  "end": 159
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "objectId2",
                  "loc": {
                    "start": 164,
                    "end": 173
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 164,
                  "end": 173
                }
              }
            ],
            "loc": {
              "start": 86,
              "end": 177
            }
          },
          "loc": {
            "start": 53,
            "end": 177
          }
        }
      ],
      "loc": {
        "start": 49,
        "end": 179
      }
    },
    "loc": {
      "start": 1,
      "end": 179
    }
  },
  "variableValues": {},
  "path": {
    "key": "reputationHistory_findOne"
  }
} as const;
