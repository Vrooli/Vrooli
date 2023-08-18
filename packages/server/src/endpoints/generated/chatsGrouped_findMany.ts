export const chatsGrouped_findMany = {
  "fieldName": "chatsGrouped",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "chatsGrouped",
        "loc": {
          "start": 58,
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
              "value": "edges",
              "loc": {
                "start": 92,
                "end": 97
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 108,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 123,
                      "end": 127
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "chatsCount",
                          "loc": {
                            "start": 142,
                            "end": 152
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 142,
                          "end": 152
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "participants",
                          "loc": {
                            "start": 165,
                            "end": 177
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "user",
                                "loc": {
                                  "start": 196,
                                  "end": 200
                                }
                              },
                              "arguments": [],
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
                                        "start": 223,
                                        "end": 225
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 223,
                                      "end": 225
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 246,
                                        "end": 257
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 246,
                                      "end": 257
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 278,
                                        "end": 284
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 278,
                                      "end": 284
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBot",
                                      "loc": {
                                        "start": 305,
                                        "end": 310
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 305,
                                      "end": 310
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 331,
                                        "end": 335
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 331,
                                      "end": 335
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 356,
                                        "end": 368
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 356,
                                      "end": 368
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 201,
                                  "end": 386
                                }
                              },
                              "loc": {
                                "start": 196,
                                "end": 386
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 403,
                                  "end": 405
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 403,
                                "end": 405
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 422,
                                  "end": 432
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 422,
                                "end": 432
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 449,
                                  "end": 459
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 449,
                                "end": 459
                              }
                            }
                          ],
                          "loc": {
                            "start": 178,
                            "end": 473
                          }
                        },
                        "loc": {
                          "start": 165,
                          "end": 473
                        }
                      }
                    ],
                    "loc": {
                      "start": 128,
                      "end": 483
                    }
                  },
                  "loc": {
                    "start": 123,
                    "end": 483
                  }
                }
              ],
              "loc": {
                "start": 98,
                "end": 489
              }
            },
            "loc": {
              "start": 92,
              "end": 489
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 494,
                "end": 502
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursor",
                    "loc": {
                      "start": 513,
                      "end": 522
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 513,
                    "end": 522
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 531,
                      "end": 542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 531,
                    "end": 542
                  }
                }
              ],
              "loc": {
                "start": 503,
                "end": 548
              }
            },
            "loc": {
              "start": 494,
              "end": 548
            }
          }
        ],
        "loc": {
          "start": 86,
          "end": 552
        }
      },
      "loc": {
        "start": 58,
        "end": 552
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
      "value": "chatsGrouped",
      "loc": {
        "start": 7,
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
              "value": "ChatsGroupedSearchInput",
              "loc": {
                "start": 28,
                "end": 51
              }
            },
            "loc": {
              "start": 28,
              "end": 51
            }
          },
          "loc": {
            "start": 28,
            "end": 52
          }
        },
        "directives": [],
        "loc": {
          "start": 20,
          "end": 52
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
            "value": "chatsGrouped",
            "loc": {
              "start": 58,
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
                  "value": "edges",
                  "loc": {
                    "start": 92,
                    "end": 97
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 108,
                          "end": 114
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 108,
                        "end": 114
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 123,
                          "end": 127
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "chatsCount",
                              "loc": {
                                "start": 142,
                                "end": 152
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 142,
                              "end": 152
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "participants",
                              "loc": {
                                "start": 165,
                                "end": 177
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "user",
                                    "loc": {
                                      "start": 196,
                                      "end": 200
                                    }
                                  },
                                  "arguments": [],
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
                                            "start": 223,
                                            "end": 225
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 223,
                                          "end": 225
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bannerImage",
                                          "loc": {
                                            "start": 246,
                                            "end": 257
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 246,
                                          "end": 257
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "handle",
                                          "loc": {
                                            "start": 278,
                                            "end": 284
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 278,
                                          "end": 284
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isBot",
                                          "loc": {
                                            "start": 305,
                                            "end": 310
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 305,
                                          "end": 310
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 331,
                                            "end": 335
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 331,
                                          "end": 335
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "profileImage",
                                          "loc": {
                                            "start": 356,
                                            "end": 368
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 356,
                                          "end": 368
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 201,
                                      "end": 386
                                    }
                                  },
                                  "loc": {
                                    "start": 196,
                                    "end": 386
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 403,
                                      "end": 405
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 403,
                                    "end": 405
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 422,
                                      "end": 432
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 422,
                                    "end": 432
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 449,
                                      "end": 459
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 449,
                                    "end": 459
                                  }
                                }
                              ],
                              "loc": {
                                "start": 178,
                                "end": 473
                              }
                            },
                            "loc": {
                              "start": 165,
                              "end": 473
                            }
                          }
                        ],
                        "loc": {
                          "start": 128,
                          "end": 483
                        }
                      },
                      "loc": {
                        "start": 123,
                        "end": 483
                      }
                    }
                  ],
                  "loc": {
                    "start": 98,
                    "end": 489
                  }
                },
                "loc": {
                  "start": 92,
                  "end": 489
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 494,
                    "end": 502
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursor",
                        "loc": {
                          "start": 513,
                          "end": 522
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 513,
                        "end": 522
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 531,
                          "end": 542
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 531,
                        "end": 542
                      }
                    }
                  ],
                  "loc": {
                    "start": 503,
                    "end": 548
                  }
                },
                "loc": {
                  "start": 494,
                  "end": 548
                }
              }
            ],
            "loc": {
              "start": 86,
              "end": 552
            }
          },
          "loc": {
            "start": 58,
            "end": 552
          }
        }
      ],
      "loc": {
        "start": 54,
        "end": 554
      }
    },
    "loc": {
      "start": 1,
      "end": 554
    }
  },
  "variableValues": {},
  "path": {
    "key": "chatsGrouped_findMany"
  }
} as const;
