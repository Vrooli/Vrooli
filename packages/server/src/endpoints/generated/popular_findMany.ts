export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 7873,
          "end": 7881
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7882,
              "end": 7887
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7890,
                "end": 7895
              }
            },
            "loc": {
              "start": 7889,
              "end": 7895
            }
          },
          "loc": {
            "start": 7882,
            "end": 7895
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
                "start": 7903,
                "end": 7908
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
                      "start": 7919,
                      "end": 7925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7919,
                    "end": 7925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 7934,
                      "end": 7938
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Api",
                            "loc": {
                              "start": 7960,
                              "end": 7963
                            }
                          },
                          "loc": {
                            "start": 7960,
                            "end": 7963
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Api_list",
                                "loc": {
                                  "start": 7985,
                                  "end": 7993
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 7982,
                                "end": 7993
                              }
                            }
                          ],
                          "loc": {
                            "start": 7964,
                            "end": 8007
                          }
                        },
                        "loc": {
                          "start": 7953,
                          "end": 8007
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Code",
                            "loc": {
                              "start": 8027,
                              "end": 8031
                            }
                          },
                          "loc": {
                            "start": 8027,
                            "end": 8031
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Code_list",
                                "loc": {
                                  "start": 8053,
                                  "end": 8062
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8050,
                                "end": 8062
                              }
                            }
                          ],
                          "loc": {
                            "start": 8032,
                            "end": 8076
                          }
                        },
                        "loc": {
                          "start": 8020,
                          "end": 8076
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Note",
                            "loc": {
                              "start": 8096,
                              "end": 8100
                            }
                          },
                          "loc": {
                            "start": 8096,
                            "end": 8100
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Note_list",
                                "loc": {
                                  "start": 8122,
                                  "end": 8131
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8119,
                                "end": 8131
                              }
                            }
                          ],
                          "loc": {
                            "start": 8101,
                            "end": 8145
                          }
                        },
                        "loc": {
                          "start": 8089,
                          "end": 8145
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 8165,
                              "end": 8172
                            }
                          },
                          "loc": {
                            "start": 8165,
                            "end": 8172
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 8194,
                                  "end": 8206
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8191,
                                "end": 8206
                              }
                            }
                          ],
                          "loc": {
                            "start": 8173,
                            "end": 8220
                          }
                        },
                        "loc": {
                          "start": 8158,
                          "end": 8220
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Question",
                            "loc": {
                              "start": 8240,
                              "end": 8248
                            }
                          },
                          "loc": {
                            "start": 8240,
                            "end": 8248
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Question_list",
                                "loc": {
                                  "start": 8270,
                                  "end": 8283
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8267,
                                "end": 8283
                              }
                            }
                          ],
                          "loc": {
                            "start": 8249,
                            "end": 8297
                          }
                        },
                        "loc": {
                          "start": 8233,
                          "end": 8297
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 8317,
                              "end": 8324
                            }
                          },
                          "loc": {
                            "start": 8317,
                            "end": 8324
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Routine_list",
                                "loc": {
                                  "start": 8346,
                                  "end": 8358
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8343,
                                "end": 8358
                              }
                            }
                          ],
                          "loc": {
                            "start": 8325,
                            "end": 8372
                          }
                        },
                        "loc": {
                          "start": 8310,
                          "end": 8372
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Standard",
                            "loc": {
                              "start": 8392,
                              "end": 8400
                            }
                          },
                          "loc": {
                            "start": 8392,
                            "end": 8400
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Standard_list",
                                "loc": {
                                  "start": 8422,
                                  "end": 8435
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8419,
                                "end": 8435
                              }
                            }
                          ],
                          "loc": {
                            "start": 8401,
                            "end": 8449
                          }
                        },
                        "loc": {
                          "start": 8385,
                          "end": 8449
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Team",
                            "loc": {
                              "start": 8469,
                              "end": 8473
                            }
                          },
                          "loc": {
                            "start": 8469,
                            "end": 8473
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Team_list",
                                "loc": {
                                  "start": 8495,
                                  "end": 8504
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8492,
                                "end": 8504
                              }
                            }
                          ],
                          "loc": {
                            "start": 8474,
                            "end": 8518
                          }
                        },
                        "loc": {
                          "start": 8462,
                          "end": 8518
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "User",
                            "loc": {
                              "start": 8538,
                              "end": 8542
                            }
                          },
                          "loc": {
                            "start": 8538,
                            "end": 8542
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "User_list",
                                "loc": {
                                  "start": 8564,
                                  "end": 8573
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8561,
                                "end": 8573
                              }
                            }
                          ],
                          "loc": {
                            "start": 8543,
                            "end": 8587
                          }
                        },
                        "loc": {
                          "start": 8531,
                          "end": 8587
                        }
                      }
                    ],
                    "loc": {
                      "start": 7939,
                      "end": 8597
                    }
                  },
                  "loc": {
                    "start": 7934,
                    "end": 8597
                  }
                }
              ],
              "loc": {
                "start": 7909,
                "end": 8603
              }
            },
            "loc": {
              "start": 7903,
              "end": 8603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8608,
                "end": 8616
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
                    "value": "hasNextPage",
                    "loc": {
                      "start": 8627,
                      "end": 8638
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8627,
                    "end": 8638
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8647,
                      "end": 8659
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8647,
                    "end": 8659
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorCode",
                    "loc": {
                      "start": 8668,
                      "end": 8681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8668,
                    "end": 8681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8690,
                      "end": 8703
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8690,
                    "end": 8703
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8712,
                      "end": 8728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8712,
                    "end": 8728
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8737,
                      "end": 8754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8737,
                    "end": 8754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8763,
                      "end": 8779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8763,
                    "end": 8779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 8788,
                      "end": 8805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8788,
                    "end": 8805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorTeam",
                    "loc": {
                      "start": 8814,
                      "end": 8827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8814,
                    "end": 8827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 8836,
                      "end": 8849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8836,
                    "end": 8849
                  }
                }
              ],
              "loc": {
                "start": 8617,
                "end": 8855
              }
            },
            "loc": {
              "start": 8608,
              "end": 8855
            }
          }
        ],
        "loc": {
          "start": 7897,
          "end": 8859
        }
      },
      "loc": {
        "start": 7873,
        "end": 8859
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
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
              "value": "translations",
              "loc": {
                "start": 43,
                "end": 55
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
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 259,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 259,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
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
                    "value": "canComment",
                    "loc": {
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 553,
                  "end": 557
                }
              },
              "loc": {
                "start": 553,
                "end": 557
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 571,
                      "end": 579
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 568,
                    "end": 579
                  }
                }
              ],
              "loc": {
                "start": 558,
                "end": 585
              }
            },
            "loc": {
              "start": 546,
              "end": 585
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 597,
                  "end": 601
                }
              },
              "loc": {
                "start": 597,
                "end": 601
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 615,
                      "end": 623
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 612,
                    "end": 623
                  }
                }
              ],
              "loc": {
                "start": 602,
                "end": 629
              }
            },
            "loc": {
              "start": 590,
              "end": 629
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 631
        }
      },
      "loc": {
        "start": 534,
        "end": 631
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 632,
          "end": 643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 632,
        "end": 643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 644,
          "end": 658
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 644,
        "end": 658
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 659,
          "end": 664
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 659,
        "end": 664
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 665,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 665,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 675,
          "end": 679
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 689,
                "end": 697
              }
            },
            "directives": [],
            "loc": {
              "start": 686,
              "end": 697
            }
          }
        ],
        "loc": {
          "start": 680,
          "end": 699
        }
      },
      "loc": {
        "start": 675,
        "end": 699
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 700,
          "end": 714
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 700,
        "end": 714
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 715,
          "end": 720
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 715,
        "end": 720
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 721,
          "end": 724
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
              "value": "canDelete",
              "loc": {
                "start": 731,
                "end": 740
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 740
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 745,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 745,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 777,
                "end": 786
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 786
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 791,
                "end": 798
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 791,
              "end": 798
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 803,
                "end": 811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 803,
              "end": 811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 816,
                "end": 828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 816,
              "end": 828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 833,
                "end": 841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 833,
              "end": 841
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 846,
                "end": 854
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 846,
              "end": 854
            }
          }
        ],
        "loc": {
          "start": 725,
          "end": 856
        }
      },
      "loc": {
        "start": 721,
        "end": 856
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 885,
          "end": 887
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 885,
        "end": 887
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 888,
          "end": 897
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 888,
        "end": 897
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 929,
          "end": 937
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
              "value": "translations",
              "loc": {
                "start": 944,
                "end": 956
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
                      "start": 967,
                      "end": 969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 967,
                    "end": 969
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 978,
                      "end": 986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 978,
                    "end": 986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 995,
                      "end": 1006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 995,
                    "end": 1006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 1015,
                      "end": 1027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1015,
                    "end": 1027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1036,
                      "end": 1040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1036,
                    "end": 1040
                  }
                }
              ],
              "loc": {
                "start": 957,
                "end": 1046
              }
            },
            "loc": {
              "start": 944,
              "end": 1046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1051,
                "end": 1053
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1051,
              "end": 1053
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1058,
                "end": 1068
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1058,
              "end": 1068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1073,
                "end": 1083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1073,
              "end": 1083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1088,
                "end": 1098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1088,
              "end": 1098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1103,
                "end": 1112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1103,
              "end": 1112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1117,
                "end": 1125
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1117,
              "end": 1125
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1130,
                "end": 1139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1130,
              "end": 1139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeLanguage",
              "loc": {
                "start": 1144,
                "end": 1156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1144,
              "end": 1156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeType",
              "loc": {
                "start": 1161,
                "end": 1169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1161,
              "end": 1169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 1174,
                "end": 1181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1174,
              "end": 1181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1186,
                "end": 1198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1203,
                "end": 1215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1203,
              "end": 1215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "calledByRoutineVersionsCount",
              "loc": {
                "start": 1220,
                "end": 1248
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1220,
              "end": 1248
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1253,
                "end": 1266
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1253,
              "end": 1266
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1271,
                "end": 1293
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1271,
              "end": 1293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1298,
                "end": 1308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1298,
              "end": 1308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1313,
                "end": 1325
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1313,
              "end": 1325
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1330,
                "end": 1333
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
                    "value": "canComment",
                    "loc": {
                      "start": 1344,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1344,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1363,
                      "end": 1370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1363,
                    "end": 1370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1379,
                      "end": 1388
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1379,
                    "end": 1388
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1397,
                      "end": 1406
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1397,
                    "end": 1406
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1415,
                      "end": 1424
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1415,
                    "end": 1424
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1433,
                      "end": 1439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1433,
                    "end": 1439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1448,
                      "end": 1455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1448,
                    "end": 1455
                  }
                }
              ],
              "loc": {
                "start": 1334,
                "end": 1461
              }
            },
            "loc": {
              "start": 1330,
              "end": 1461
            }
          }
        ],
        "loc": {
          "start": 938,
          "end": 1463
        }
      },
      "loc": {
        "start": 929,
        "end": 1463
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1464,
          "end": 1466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1464,
        "end": 1466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1467,
          "end": 1477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1467,
        "end": 1477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1478,
          "end": 1488
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1478,
        "end": 1488
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1489,
          "end": 1498
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1489,
        "end": 1498
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1499,
          "end": 1510
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1499,
        "end": 1510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1511,
          "end": 1517
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1527,
                "end": 1537
              }
            },
            "directives": [],
            "loc": {
              "start": 1524,
              "end": 1537
            }
          }
        ],
        "loc": {
          "start": 1518,
          "end": 1539
        }
      },
      "loc": {
        "start": 1511,
        "end": 1539
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1540,
          "end": 1545
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 1559,
                  "end": 1563
                }
              },
              "loc": {
                "start": 1559,
                "end": 1563
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 1577,
                      "end": 1585
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1574,
                    "end": 1585
                  }
                }
              ],
              "loc": {
                "start": 1564,
                "end": 1591
              }
            },
            "loc": {
              "start": 1552,
              "end": 1591
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1603,
                  "end": 1607
                }
              },
              "loc": {
                "start": 1603,
                "end": 1607
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1621,
                      "end": 1629
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1618,
                    "end": 1629
                  }
                }
              ],
              "loc": {
                "start": 1608,
                "end": 1635
              }
            },
            "loc": {
              "start": 1596,
              "end": 1635
            }
          }
        ],
        "loc": {
          "start": 1546,
          "end": 1637
        }
      },
      "loc": {
        "start": 1540,
        "end": 1637
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1638,
          "end": 1649
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1638,
        "end": 1649
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1650,
          "end": 1664
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1650,
        "end": 1664
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1665,
          "end": 1670
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1665,
        "end": 1670
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1671,
          "end": 1680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1671,
        "end": 1680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1681,
          "end": 1685
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1695,
                "end": 1703
              }
            },
            "directives": [],
            "loc": {
              "start": 1692,
              "end": 1703
            }
          }
        ],
        "loc": {
          "start": 1686,
          "end": 1705
        }
      },
      "loc": {
        "start": 1681,
        "end": 1705
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1706,
          "end": 1720
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1706,
        "end": 1720
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1721,
          "end": 1726
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1721,
        "end": 1726
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1727,
          "end": 1730
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
              "value": "canDelete",
              "loc": {
                "start": 1737,
                "end": 1746
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1737,
              "end": 1746
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1751,
                "end": 1762
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1751,
              "end": 1762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1767,
                "end": 1778
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1767,
              "end": 1778
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1783,
                "end": 1792
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1783,
              "end": 1792
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1797,
                "end": 1804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1797,
              "end": 1804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1809,
                "end": 1817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1809,
              "end": 1817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1822,
                "end": 1834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1822,
              "end": 1834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1839,
                "end": 1847
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1839,
              "end": 1847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1852,
                "end": 1860
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1852,
              "end": 1860
            }
          }
        ],
        "loc": {
          "start": 1731,
          "end": 1862
        }
      },
      "loc": {
        "start": 1727,
        "end": 1862
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1893,
          "end": 1895
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1893,
        "end": 1895
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1896,
          "end": 1905
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1896,
        "end": 1905
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1939,
          "end": 1941
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1939,
        "end": 1941
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1942,
          "end": 1952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1942,
        "end": 1952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1953,
          "end": 1963
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1953,
        "end": 1963
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1964,
          "end": 1969
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1964,
        "end": 1969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1970,
          "end": 1975
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1970,
        "end": 1975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1976,
          "end": 1981
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 1995,
                  "end": 1999
                }
              },
              "loc": {
                "start": 1995,
                "end": 1999
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 2013,
                      "end": 2021
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2010,
                    "end": 2021
                  }
                }
              ],
              "loc": {
                "start": 2000,
                "end": 2027
              }
            },
            "loc": {
              "start": 1988,
              "end": 2027
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 2039,
                  "end": 2043
                }
              },
              "loc": {
                "start": 2039,
                "end": 2043
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 2057,
                      "end": 2065
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2054,
                    "end": 2065
                  }
                }
              ],
              "loc": {
                "start": 2044,
                "end": 2071
              }
            },
            "loc": {
              "start": 2032,
              "end": 2071
            }
          }
        ],
        "loc": {
          "start": 1982,
          "end": 2073
        }
      },
      "loc": {
        "start": 1976,
        "end": 2073
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2074,
          "end": 2077
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
              "value": "canDelete",
              "loc": {
                "start": 2084,
                "end": 2093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2084,
              "end": 2093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2098,
                "end": 2107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2098,
              "end": 2107
            }
          }
        ],
        "loc": {
          "start": 2078,
          "end": 2109
        }
      },
      "loc": {
        "start": 2074,
        "end": 2109
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2141,
          "end": 2149
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
              "value": "translations",
              "loc": {
                "start": 2156,
                "end": 2168
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
                      "start": 2179,
                      "end": 2181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2179,
                    "end": 2181
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2190,
                      "end": 2198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2190,
                    "end": 2198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2207,
                      "end": 2218
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2207,
                    "end": 2218
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2227,
                      "end": 2231
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2227,
                    "end": 2231
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 2240,
                      "end": 2245
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
                            "start": 2260,
                            "end": 2262
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2260,
                          "end": 2262
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 2275,
                            "end": 2284
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2275,
                          "end": 2284
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 2297,
                            "end": 2301
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2297,
                          "end": 2301
                        }
                      }
                    ],
                    "loc": {
                      "start": 2246,
                      "end": 2311
                    }
                  },
                  "loc": {
                    "start": 2240,
                    "end": 2311
                  }
                }
              ],
              "loc": {
                "start": 2169,
                "end": 2317
              }
            },
            "loc": {
              "start": 2156,
              "end": 2317
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2322,
                "end": 2324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2322,
              "end": 2324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2329,
                "end": 2339
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2329,
              "end": 2339
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2344,
                "end": 2354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2344,
              "end": 2354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 2359,
                "end": 2367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2359,
              "end": 2367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2372,
                "end": 2381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2372,
              "end": 2381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2386,
                "end": 2398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2386,
              "end": 2398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 2403,
                "end": 2415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2403,
              "end": 2415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 2420,
                "end": 2432
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2420,
              "end": 2432
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2437,
                "end": 2440
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
                    "value": "canComment",
                    "loc": {
                      "start": 2451,
                      "end": 2461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2451,
                    "end": 2461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 2470,
                      "end": 2477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2470,
                    "end": 2477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2486,
                      "end": 2495
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2486,
                    "end": 2495
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2504,
                      "end": 2513
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2504,
                    "end": 2513
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2522,
                      "end": 2531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2522,
                    "end": 2531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 2540,
                      "end": 2546
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2540,
                    "end": 2546
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2555,
                      "end": 2562
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2555,
                    "end": 2562
                  }
                }
              ],
              "loc": {
                "start": 2441,
                "end": 2568
              }
            },
            "loc": {
              "start": 2437,
              "end": 2568
            }
          }
        ],
        "loc": {
          "start": 2150,
          "end": 2570
        }
      },
      "loc": {
        "start": 2141,
        "end": 2570
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2571,
          "end": 2573
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2571,
        "end": 2573
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2574,
          "end": 2584
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2574,
        "end": 2584
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2585,
          "end": 2595
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2585,
        "end": 2595
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2596,
          "end": 2605
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2596,
        "end": 2605
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 2606,
          "end": 2617
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2606,
        "end": 2617
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2618,
          "end": 2624
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 2634,
                "end": 2644
              }
            },
            "directives": [],
            "loc": {
              "start": 2631,
              "end": 2644
            }
          }
        ],
        "loc": {
          "start": 2625,
          "end": 2646
        }
      },
      "loc": {
        "start": 2618,
        "end": 2646
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 2647,
          "end": 2652
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 2666,
                  "end": 2670
                }
              },
              "loc": {
                "start": 2666,
                "end": 2670
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 2684,
                      "end": 2692
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2681,
                    "end": 2692
                  }
                }
              ],
              "loc": {
                "start": 2671,
                "end": 2698
              }
            },
            "loc": {
              "start": 2659,
              "end": 2698
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 2710,
                  "end": 2714
                }
              },
              "loc": {
                "start": 2710,
                "end": 2714
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 2728,
                      "end": 2736
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2725,
                    "end": 2736
                  }
                }
              ],
              "loc": {
                "start": 2715,
                "end": 2742
              }
            },
            "loc": {
              "start": 2703,
              "end": 2742
            }
          }
        ],
        "loc": {
          "start": 2653,
          "end": 2744
        }
      },
      "loc": {
        "start": 2647,
        "end": 2744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2745,
          "end": 2756
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2745,
        "end": 2756
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2757,
          "end": 2771
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2757,
        "end": 2771
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2772,
          "end": 2777
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2772,
        "end": 2777
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2778,
          "end": 2787
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2778,
        "end": 2787
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2788,
          "end": 2792
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2802,
                "end": 2810
              }
            },
            "directives": [],
            "loc": {
              "start": 2799,
              "end": 2810
            }
          }
        ],
        "loc": {
          "start": 2793,
          "end": 2812
        }
      },
      "loc": {
        "start": 2788,
        "end": 2812
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2813,
          "end": 2827
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2813,
        "end": 2827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2828,
          "end": 2833
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2828,
        "end": 2833
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2834,
          "end": 2837
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
              "value": "canDelete",
              "loc": {
                "start": 2844,
                "end": 2853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2844,
              "end": 2853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2858,
                "end": 2869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2858,
              "end": 2869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2874,
                "end": 2885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2874,
              "end": 2885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2890,
                "end": 2899
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2890,
              "end": 2899
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2904,
                "end": 2911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2904,
              "end": 2911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2916,
                "end": 2924
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2916,
              "end": 2924
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2929,
                "end": 2941
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2929,
              "end": 2941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2946,
                "end": 2954
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2946,
              "end": 2954
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2959,
                "end": 2967
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2959,
              "end": 2967
            }
          }
        ],
        "loc": {
          "start": 2838,
          "end": 2969
        }
      },
      "loc": {
        "start": 2834,
        "end": 2969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3000,
          "end": 3002
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3000,
        "end": 3002
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3003,
          "end": 3012
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3003,
        "end": 3012
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3050,
          "end": 3058
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
              "value": "translations",
              "loc": {
                "start": 3065,
                "end": 3077
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
                      "start": 3088,
                      "end": 3090
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3088,
                    "end": 3090
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3099,
                      "end": 3107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3099,
                    "end": 3107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3116,
                      "end": 3127
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3116,
                    "end": 3127
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3136,
                      "end": 3140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3136,
                    "end": 3140
                  }
                }
              ],
              "loc": {
                "start": 3078,
                "end": 3146
              }
            },
            "loc": {
              "start": 3065,
              "end": 3146
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3151,
                "end": 3153
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3151,
              "end": 3153
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3158,
                "end": 3168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3158,
              "end": 3168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3173,
                "end": 3183
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3173,
              "end": 3183
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3188,
                "end": 3204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3188,
              "end": 3204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3209,
                "end": 3217
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3209,
              "end": 3217
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3222,
                "end": 3231
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3222,
              "end": 3231
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3236,
                "end": 3248
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3236,
              "end": 3248
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3253,
                "end": 3269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3253,
              "end": 3269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3274,
                "end": 3284
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3274,
              "end": 3284
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3289,
                "end": 3301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3289,
              "end": 3301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3306,
                "end": 3318
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3306,
              "end": 3318
            }
          }
        ],
        "loc": {
          "start": 3059,
          "end": 3320
        }
      },
      "loc": {
        "start": 3050,
        "end": 3320
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3321,
          "end": 3323
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3321,
        "end": 3323
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3324,
          "end": 3334
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3324,
        "end": 3334
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3335,
          "end": 3345
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3335,
        "end": 3345
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3346,
          "end": 3355
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3346,
        "end": 3355
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3356,
          "end": 3367
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3356,
        "end": 3367
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3368,
          "end": 3374
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 3384,
                "end": 3394
              }
            },
            "directives": [],
            "loc": {
              "start": 3381,
              "end": 3394
            }
          }
        ],
        "loc": {
          "start": 3375,
          "end": 3396
        }
      },
      "loc": {
        "start": 3368,
        "end": 3396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3397,
          "end": 3402
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 3416,
                  "end": 3420
                }
              },
              "loc": {
                "start": 3416,
                "end": 3420
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 3434,
                      "end": 3442
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3431,
                    "end": 3442
                  }
                }
              ],
              "loc": {
                "start": 3421,
                "end": 3448
              }
            },
            "loc": {
              "start": 3409,
              "end": 3448
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 3460,
                  "end": 3464
                }
              },
              "loc": {
                "start": 3460,
                "end": 3464
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 3478,
                      "end": 3486
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3475,
                    "end": 3486
                  }
                }
              ],
              "loc": {
                "start": 3465,
                "end": 3492
              }
            },
            "loc": {
              "start": 3453,
              "end": 3492
            }
          }
        ],
        "loc": {
          "start": 3403,
          "end": 3494
        }
      },
      "loc": {
        "start": 3397,
        "end": 3494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3495,
          "end": 3506
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3495,
        "end": 3506
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3507,
          "end": 3521
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3507,
        "end": 3521
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3522,
          "end": 3527
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3522,
        "end": 3527
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3528,
          "end": 3537
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3528,
        "end": 3537
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3538,
          "end": 3542
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 3552,
                "end": 3560
              }
            },
            "directives": [],
            "loc": {
              "start": 3549,
              "end": 3560
            }
          }
        ],
        "loc": {
          "start": 3543,
          "end": 3562
        }
      },
      "loc": {
        "start": 3538,
        "end": 3562
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3563,
          "end": 3577
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3563,
        "end": 3577
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3578,
          "end": 3583
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3578,
        "end": 3583
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3584,
          "end": 3587
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
              "value": "canDelete",
              "loc": {
                "start": 3594,
                "end": 3603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3594,
              "end": 3603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3608,
                "end": 3619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3608,
              "end": 3619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3624,
                "end": 3635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3624,
              "end": 3635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3640,
                "end": 3649
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3640,
              "end": 3649
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3654,
                "end": 3661
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3654,
              "end": 3661
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3666,
                "end": 3674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3666,
              "end": 3674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3679,
                "end": 3691
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3679,
              "end": 3691
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3696,
                "end": 3704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3696,
              "end": 3704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3709,
                "end": 3717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3709,
              "end": 3717
            }
          }
        ],
        "loc": {
          "start": 3588,
          "end": 3719
        }
      },
      "loc": {
        "start": 3584,
        "end": 3719
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3756,
          "end": 3758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3756,
        "end": 3758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3759,
          "end": 3768
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3759,
        "end": 3768
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3808,
          "end": 3820
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
                "start": 3827,
                "end": 3829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3827,
              "end": 3829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3834,
                "end": 3842
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3834,
              "end": 3842
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3847,
                "end": 3858
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3847,
              "end": 3858
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3863,
                "end": 3867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3863,
              "end": 3867
            }
          }
        ],
        "loc": {
          "start": 3821,
          "end": 3869
        }
      },
      "loc": {
        "start": 3808,
        "end": 3869
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3870,
          "end": 3872
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3870,
        "end": 3872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3873,
          "end": 3883
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3873,
        "end": 3883
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3884,
          "end": 3894
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3884,
        "end": 3894
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3895,
          "end": 3904
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
                "start": 3911,
                "end": 3913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3911,
              "end": 3913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3918,
                "end": 3928
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3918,
              "end": 3928
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3933,
                "end": 3943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3933,
              "end": 3943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3948,
                "end": 3959
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3948,
              "end": 3959
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3964,
                "end": 3970
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3964,
              "end": 3970
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3975,
                "end": 3980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3975,
              "end": 3980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 3985,
                "end": 4005
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3985,
              "end": 4005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 4010,
                "end": 4014
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4010,
              "end": 4014
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 4019,
                "end": 4031
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4019,
              "end": 4031
            }
          }
        ],
        "loc": {
          "start": 3905,
          "end": 4033
        }
      },
      "loc": {
        "start": 3895,
        "end": 4033
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 4034,
          "end": 4051
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4034,
        "end": 4051
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4052,
          "end": 4061
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4052,
        "end": 4061
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4062,
          "end": 4067
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4062,
        "end": 4067
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4068,
          "end": 4077
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4068,
        "end": 4077
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4078,
          "end": 4090
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4078,
        "end": 4090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4091,
          "end": 4104
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4091,
        "end": 4104
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4105,
          "end": 4117
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4105,
        "end": 4117
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4118,
          "end": 4127
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Api",
                "loc": {
                  "start": 4141,
                  "end": 4144
                }
              },
              "loc": {
                "start": 4141,
                "end": 4144
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_nav",
                    "loc": {
                      "start": 4158,
                      "end": 4165
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4155,
                    "end": 4165
                  }
                }
              ],
              "loc": {
                "start": 4145,
                "end": 4171
              }
            },
            "loc": {
              "start": 4134,
              "end": 4171
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Code",
                "loc": {
                  "start": 4183,
                  "end": 4187
                }
              },
              "loc": {
                "start": 4183,
                "end": 4187
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Code_nav",
                    "loc": {
                      "start": 4201,
                      "end": 4209
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4198,
                    "end": 4209
                  }
                }
              ],
              "loc": {
                "start": 4188,
                "end": 4215
              }
            },
            "loc": {
              "start": 4176,
              "end": 4215
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Note",
                "loc": {
                  "start": 4227,
                  "end": 4231
                }
              },
              "loc": {
                "start": 4227,
                "end": 4231
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_nav",
                    "loc": {
                      "start": 4245,
                      "end": 4253
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4242,
                    "end": 4253
                  }
                }
              ],
              "loc": {
                "start": 4232,
                "end": 4259
              }
            },
            "loc": {
              "start": 4220,
              "end": 4259
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Project",
                "loc": {
                  "start": 4271,
                  "end": 4278
                }
              },
              "loc": {
                "start": 4271,
                "end": 4278
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_nav",
                    "loc": {
                      "start": 4292,
                      "end": 4303
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4289,
                    "end": 4303
                  }
                }
              ],
              "loc": {
                "start": 4279,
                "end": 4309
              }
            },
            "loc": {
              "start": 4264,
              "end": 4309
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Routine",
                "loc": {
                  "start": 4321,
                  "end": 4328
                }
              },
              "loc": {
                "start": 4321,
                "end": 4328
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_nav",
                    "loc": {
                      "start": 4342,
                      "end": 4353
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4339,
                    "end": 4353
                  }
                }
              ],
              "loc": {
                "start": 4329,
                "end": 4359
              }
            },
            "loc": {
              "start": 4314,
              "end": 4359
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Standard",
                "loc": {
                  "start": 4371,
                  "end": 4379
                }
              },
              "loc": {
                "start": 4371,
                "end": 4379
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_nav",
                    "loc": {
                      "start": 4393,
                      "end": 4405
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4390,
                    "end": 4405
                  }
                }
              ],
              "loc": {
                "start": 4380,
                "end": 4411
              }
            },
            "loc": {
              "start": 4364,
              "end": 4411
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 4423,
                  "end": 4427
                }
              },
              "loc": {
                "start": 4423,
                "end": 4427
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 4441,
                      "end": 4449
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4438,
                    "end": 4449
                  }
                }
              ],
              "loc": {
                "start": 4428,
                "end": 4455
              }
            },
            "loc": {
              "start": 4416,
              "end": 4455
            }
          }
        ],
        "loc": {
          "start": 4128,
          "end": 4457
        }
      },
      "loc": {
        "start": 4118,
        "end": 4457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4458,
          "end": 4462
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 4472,
                "end": 4480
              }
            },
            "directives": [],
            "loc": {
              "start": 4469,
              "end": 4480
            }
          }
        ],
        "loc": {
          "start": 4463,
          "end": 4482
        }
      },
      "loc": {
        "start": 4458,
        "end": 4482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4483,
          "end": 4486
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
              "value": "reaction",
              "loc": {
                "start": 4493,
                "end": 4501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4493,
              "end": 4501
            }
          }
        ],
        "loc": {
          "start": 4487,
          "end": 4503
        }
      },
      "loc": {
        "start": 4483,
        "end": 4503
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4541,
          "end": 4549
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
              "value": "translations",
              "loc": {
                "start": 4556,
                "end": 4568
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
                      "start": 4579,
                      "end": 4581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4579,
                    "end": 4581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4590,
                      "end": 4598
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4590,
                    "end": 4598
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4607,
                      "end": 4618
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4607,
                    "end": 4618
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4627,
                      "end": 4639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4627,
                    "end": 4639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4648,
                      "end": 4652
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4648,
                    "end": 4652
                  }
                }
              ],
              "loc": {
                "start": 4569,
                "end": 4658
              }
            },
            "loc": {
              "start": 4556,
              "end": 4658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4663,
                "end": 4665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4663,
              "end": 4665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4670,
                "end": 4680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4670,
              "end": 4680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4685,
                "end": 4695
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4685,
              "end": 4695
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4700,
                "end": 4711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4700,
              "end": 4711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4716,
                "end": 4729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4716,
              "end": 4729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4734,
                "end": 4744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4734,
              "end": 4744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4749,
                "end": 4758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4749,
              "end": 4758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4763,
                "end": 4771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4763,
              "end": 4771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4776,
                "end": 4785
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4776,
              "end": 4785
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineType",
              "loc": {
                "start": 4790,
                "end": 4801
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4790,
              "end": 4801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4806,
                "end": 4816
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4806,
              "end": 4816
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4821,
                "end": 4833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4821,
              "end": 4833
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4838,
                "end": 4852
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4838,
              "end": 4852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4857,
                "end": 4869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4857,
              "end": 4869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4874,
                "end": 4886
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4874,
              "end": 4886
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4891,
                "end": 4904
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4891,
              "end": 4904
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4909,
                "end": 4931
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4909,
              "end": 4931
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 4936,
                "end": 4946
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4936,
              "end": 4946
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 4951,
                "end": 4962
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4951,
              "end": 4962
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 4967,
                "end": 4977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4967,
              "end": 4977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4982,
                "end": 4996
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4982,
              "end": 4996
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 5001,
                "end": 5013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5001,
              "end": 5013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5018,
                "end": 5030
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5018,
              "end": 5030
            }
          }
        ],
        "loc": {
          "start": 4550,
          "end": 5032
        }
      },
      "loc": {
        "start": 4541,
        "end": 5032
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5033,
          "end": 5035
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5033,
        "end": 5035
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5036,
          "end": 5046
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5036,
        "end": 5046
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5047,
          "end": 5057
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5047,
        "end": 5057
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5058,
          "end": 5068
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5058,
        "end": 5068
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5069,
          "end": 5078
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5069,
        "end": 5078
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5079,
          "end": 5090
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5079,
        "end": 5090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5091,
          "end": 5097
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 5107,
                "end": 5117
              }
            },
            "directives": [],
            "loc": {
              "start": 5104,
              "end": 5117
            }
          }
        ],
        "loc": {
          "start": 5098,
          "end": 5119
        }
      },
      "loc": {
        "start": 5091,
        "end": 5119
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5120,
          "end": 5125
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 5139,
                  "end": 5143
                }
              },
              "loc": {
                "start": 5139,
                "end": 5143
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 5157,
                      "end": 5165
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5154,
                    "end": 5165
                  }
                }
              ],
              "loc": {
                "start": 5144,
                "end": 5171
              }
            },
            "loc": {
              "start": 5132,
              "end": 5171
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 5183,
                  "end": 5187
                }
              },
              "loc": {
                "start": 5183,
                "end": 5187
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5201,
                      "end": 5209
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5198,
                    "end": 5209
                  }
                }
              ],
              "loc": {
                "start": 5188,
                "end": 5215
              }
            },
            "loc": {
              "start": 5176,
              "end": 5215
            }
          }
        ],
        "loc": {
          "start": 5126,
          "end": 5217
        }
      },
      "loc": {
        "start": 5120,
        "end": 5217
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5218,
          "end": 5229
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5218,
        "end": 5229
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5230,
          "end": 5244
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5230,
        "end": 5244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5245,
          "end": 5250
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5245,
        "end": 5250
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5251,
          "end": 5260
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5251,
        "end": 5260
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5261,
          "end": 5265
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 5275,
                "end": 5283
              }
            },
            "directives": [],
            "loc": {
              "start": 5272,
              "end": 5283
            }
          }
        ],
        "loc": {
          "start": 5266,
          "end": 5285
        }
      },
      "loc": {
        "start": 5261,
        "end": 5285
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5286,
          "end": 5300
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5286,
        "end": 5300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5301,
          "end": 5306
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5301,
        "end": 5306
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5307,
          "end": 5310
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
              "value": "canComment",
              "loc": {
                "start": 5317,
                "end": 5327
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5317,
              "end": 5327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5332,
                "end": 5341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5332,
              "end": 5341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5346,
                "end": 5357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5346,
              "end": 5357
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5362,
                "end": 5371
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5362,
              "end": 5371
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5376,
                "end": 5383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5376,
              "end": 5383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5388,
                "end": 5396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5388,
              "end": 5396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5401,
                "end": 5413
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5401,
              "end": 5413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5418,
                "end": 5426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5418,
              "end": 5426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5431,
                "end": 5439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5431,
              "end": 5439
            }
          }
        ],
        "loc": {
          "start": 5311,
          "end": 5441
        }
      },
      "loc": {
        "start": 5307,
        "end": 5441
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5478,
          "end": 5480
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5478,
        "end": 5480
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5481,
          "end": 5491
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5481,
        "end": 5491
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5492,
          "end": 5501
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5492,
        "end": 5501
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5541,
          "end": 5549
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
              "value": "translations",
              "loc": {
                "start": 5556,
                "end": 5568
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
                      "start": 5579,
                      "end": 5581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5579,
                    "end": 5581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5590,
                      "end": 5598
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5590,
                    "end": 5598
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5607,
                      "end": 5618
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5607,
                    "end": 5618
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5627,
                      "end": 5639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5627,
                    "end": 5639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5648,
                      "end": 5652
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5648,
                    "end": 5652
                  }
                }
              ],
              "loc": {
                "start": 5569,
                "end": 5658
              }
            },
            "loc": {
              "start": 5556,
              "end": 5658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5663,
                "end": 5665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5663,
              "end": 5665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5670,
                "end": 5680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5670,
              "end": 5680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5685,
                "end": 5695
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5685,
              "end": 5695
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5700,
                "end": 5710
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5700,
              "end": 5710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 5715,
                "end": 5721
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5715,
              "end": 5721
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5726,
                "end": 5734
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5726,
              "end": 5734
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5739,
                "end": 5748
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5739,
              "end": 5748
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5753,
                "end": 5760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5753,
              "end": 5760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 5765,
                "end": 5777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5765,
              "end": 5777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 5782,
                "end": 5787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5782,
              "end": 5787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 5792,
                "end": 5795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5792,
              "end": 5795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5800,
                "end": 5812
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5800,
              "end": 5812
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5817,
                "end": 5829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5817,
              "end": 5829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5834,
                "end": 5847
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5834,
              "end": 5847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5852,
                "end": 5874
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5852,
              "end": 5874
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5879,
                "end": 5889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5879,
              "end": 5889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5894,
                "end": 5906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5894,
              "end": 5906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5911,
                "end": 5914
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
                    "value": "canComment",
                    "loc": {
                      "start": 5925,
                      "end": 5935
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5925,
                    "end": 5935
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5944,
                      "end": 5951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5944,
                    "end": 5951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5960,
                      "end": 5969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5960,
                    "end": 5969
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5978,
                      "end": 5987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5978,
                    "end": 5987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5996,
                      "end": 6005
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5996,
                    "end": 6005
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6014,
                      "end": 6020
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6014,
                    "end": 6020
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6029,
                      "end": 6036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6029,
                    "end": 6036
                  }
                }
              ],
              "loc": {
                "start": 5915,
                "end": 6042
              }
            },
            "loc": {
              "start": 5911,
              "end": 6042
            }
          }
        ],
        "loc": {
          "start": 5550,
          "end": 6044
        }
      },
      "loc": {
        "start": 5541,
        "end": 6044
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6045,
          "end": 6047
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6045,
        "end": 6047
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6048,
          "end": 6058
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6048,
        "end": 6058
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6059,
          "end": 6069
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6059,
        "end": 6069
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6070,
          "end": 6079
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6070,
        "end": 6079
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6080,
          "end": 6091
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6080,
        "end": 6091
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6092,
          "end": 6098
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 6108,
                "end": 6118
              }
            },
            "directives": [],
            "loc": {
              "start": 6105,
              "end": 6118
            }
          }
        ],
        "loc": {
          "start": 6099,
          "end": 6120
        }
      },
      "loc": {
        "start": 6092,
        "end": 6120
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6121,
          "end": 6126
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 6140,
                  "end": 6144
                }
              },
              "loc": {
                "start": 6140,
                "end": 6144
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 6158,
                      "end": 6166
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6155,
                    "end": 6166
                  }
                }
              ],
              "loc": {
                "start": 6145,
                "end": 6172
              }
            },
            "loc": {
              "start": 6133,
              "end": 6172
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 6184,
                  "end": 6188
                }
              },
              "loc": {
                "start": 6184,
                "end": 6188
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6202,
                      "end": 6210
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6199,
                    "end": 6210
                  }
                }
              ],
              "loc": {
                "start": 6189,
                "end": 6216
              }
            },
            "loc": {
              "start": 6177,
              "end": 6216
            }
          }
        ],
        "loc": {
          "start": 6127,
          "end": 6218
        }
      },
      "loc": {
        "start": 6121,
        "end": 6218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6219,
          "end": 6230
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6219,
        "end": 6230
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6231,
          "end": 6245
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6231,
        "end": 6245
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6246,
          "end": 6251
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6246,
        "end": 6251
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6252,
          "end": 6261
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6252,
        "end": 6261
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6262,
          "end": 6266
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6276,
                "end": 6284
              }
            },
            "directives": [],
            "loc": {
              "start": 6273,
              "end": 6284
            }
          }
        ],
        "loc": {
          "start": 6267,
          "end": 6286
        }
      },
      "loc": {
        "start": 6262,
        "end": 6286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6287,
          "end": 6301
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6287,
        "end": 6301
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6302,
          "end": 6307
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6302,
        "end": 6307
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6308,
          "end": 6311
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
              "value": "canDelete",
              "loc": {
                "start": 6318,
                "end": 6327
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6318,
              "end": 6327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6332,
                "end": 6343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6332,
              "end": 6343
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6348,
                "end": 6359
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6348,
              "end": 6359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6364,
                "end": 6373
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6364,
              "end": 6373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6378,
                "end": 6385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6378,
              "end": 6385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6390,
                "end": 6398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6390,
              "end": 6398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6403,
                "end": 6415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6403,
              "end": 6415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6420,
                "end": 6428
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6420,
              "end": 6428
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6433,
                "end": 6441
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6433,
              "end": 6441
            }
          }
        ],
        "loc": {
          "start": 6312,
          "end": 6443
        }
      },
      "loc": {
        "start": 6308,
        "end": 6443
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6482,
          "end": 6484
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6482,
        "end": 6484
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6485,
          "end": 6494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6485,
        "end": 6494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6524,
          "end": 6526
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6524,
        "end": 6526
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6527,
          "end": 6537
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6527,
        "end": 6537
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6538,
          "end": 6541
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6538,
        "end": 6541
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6542,
          "end": 6551
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6542,
        "end": 6551
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6552,
          "end": 6564
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
                "start": 6571,
                "end": 6573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6571,
              "end": 6573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6578,
                "end": 6586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6578,
              "end": 6586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6591,
                "end": 6602
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6591,
              "end": 6602
            }
          }
        ],
        "loc": {
          "start": 6565,
          "end": 6604
        }
      },
      "loc": {
        "start": 6552,
        "end": 6604
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6605,
          "end": 6608
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
              "value": "isOwn",
              "loc": {
                "start": 6615,
                "end": 6620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6615,
              "end": 6620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6625,
                "end": 6637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6625,
              "end": 6637
            }
          }
        ],
        "loc": {
          "start": 6609,
          "end": 6639
        }
      },
      "loc": {
        "start": 6605,
        "end": 6639
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6671,
          "end": 6673
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6671,
        "end": 6673
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 6674,
          "end": 6685
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6674,
        "end": 6685
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 6686,
          "end": 6692
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6686,
        "end": 6692
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6693,
          "end": 6703
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6693,
        "end": 6703
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6704,
          "end": 6714
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6704,
        "end": 6714
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 6715,
          "end": 6733
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6715,
        "end": 6733
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6734,
          "end": 6743
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6734,
        "end": 6743
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 6744,
          "end": 6757
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6744,
        "end": 6757
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 6758,
          "end": 6770
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6758,
        "end": 6770
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 6771,
          "end": 6783
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6771,
        "end": 6783
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 6784,
          "end": 6796
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6784,
        "end": 6796
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6797,
          "end": 6806
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6797,
        "end": 6806
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6807,
          "end": 6811
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6821,
                "end": 6829
              }
            },
            "directives": [],
            "loc": {
              "start": 6818,
              "end": 6829
            }
          }
        ],
        "loc": {
          "start": 6812,
          "end": 6831
        }
      },
      "loc": {
        "start": 6807,
        "end": 6831
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6832,
          "end": 6844
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
                "start": 6851,
                "end": 6853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6851,
              "end": 6853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6858,
                "end": 6866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6858,
              "end": 6866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 6871,
                "end": 6874
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6871,
              "end": 6874
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6879,
                "end": 6883
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6879,
              "end": 6883
            }
          }
        ],
        "loc": {
          "start": 6845,
          "end": 6885
        }
      },
      "loc": {
        "start": 6832,
        "end": 6885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6886,
          "end": 6889
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
              "value": "canAddMembers",
              "loc": {
                "start": 6896,
                "end": 6909
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6896,
              "end": 6909
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6914,
                "end": 6923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6914,
              "end": 6923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6928,
                "end": 6939
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6928,
              "end": 6939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 6944,
                "end": 6953
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6944,
              "end": 6953
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6958,
                "end": 6967
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6958,
              "end": 6967
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6972,
                "end": 6979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6972,
              "end": 6979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6984,
                "end": 6996
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6984,
              "end": 6996
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7001,
                "end": 7009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7001,
              "end": 7009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7014,
                "end": 7028
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
                      "start": 7039,
                      "end": 7041
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7039,
                    "end": 7041
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7050,
                      "end": 7060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7050,
                    "end": 7060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7069,
                      "end": 7079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7069,
                    "end": 7079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7088,
                      "end": 7095
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7088,
                    "end": 7095
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7104,
                      "end": 7115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7104,
                    "end": 7115
                  }
                }
              ],
              "loc": {
                "start": 7029,
                "end": 7121
              }
            },
            "loc": {
              "start": 7014,
              "end": 7121
            }
          }
        ],
        "loc": {
          "start": 6890,
          "end": 7123
        }
      },
      "loc": {
        "start": 6886,
        "end": 7123
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7154,
          "end": 7156
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7154,
        "end": 7156
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7157,
          "end": 7168
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7157,
        "end": 7168
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7169,
          "end": 7175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7169,
        "end": 7175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7176,
          "end": 7188
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7176,
        "end": 7188
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7189,
          "end": 7192
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
              "value": "canAddMembers",
              "loc": {
                "start": 7199,
                "end": 7212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7199,
              "end": 7212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7217,
                "end": 7226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7217,
              "end": 7226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7231,
                "end": 7242
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7231,
              "end": 7242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7247,
                "end": 7256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7247,
              "end": 7256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7261,
                "end": 7270
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7261,
              "end": 7270
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7275,
                "end": 7282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7275,
              "end": 7282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7287,
                "end": 7299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7287,
              "end": 7299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7304,
                "end": 7312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7304,
              "end": 7312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7317,
                "end": 7331
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
                      "start": 7342,
                      "end": 7344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7342,
                    "end": 7344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7353,
                      "end": 7363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7353,
                    "end": 7363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7372,
                      "end": 7382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7372,
                    "end": 7382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7391,
                      "end": 7398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7391,
                    "end": 7398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7407,
                      "end": 7418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7407,
                    "end": 7418
                  }
                }
              ],
              "loc": {
                "start": 7332,
                "end": 7424
              }
            },
            "loc": {
              "start": 7317,
              "end": 7424
            }
          }
        ],
        "loc": {
          "start": 7193,
          "end": 7426
        }
      },
      "loc": {
        "start": 7189,
        "end": 7426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7458,
          "end": 7470
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
                "start": 7477,
                "end": 7479
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7477,
              "end": 7479
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7484,
                "end": 7492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7484,
              "end": 7492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7497,
                "end": 7500
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7497,
              "end": 7500
            }
          }
        ],
        "loc": {
          "start": 7471,
          "end": 7502
        }
      },
      "loc": {
        "start": 7458,
        "end": 7502
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7503,
          "end": 7505
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7503,
        "end": 7505
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7506,
          "end": 7516
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7506,
        "end": 7516
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7517,
          "end": 7527
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7517,
        "end": 7527
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7528,
          "end": 7539
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7528,
        "end": 7539
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7540,
          "end": 7546
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7540,
        "end": 7546
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7547,
          "end": 7552
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7547,
        "end": 7552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7553,
          "end": 7573
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7553,
        "end": 7573
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7574,
          "end": 7578
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7574,
        "end": 7578
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7579,
          "end": 7591
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7579,
        "end": 7591
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7592,
          "end": 7601
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7592,
        "end": 7601
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7602,
          "end": 7622
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7602,
        "end": 7622
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7623,
          "end": 7626
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
              "value": "canDelete",
              "loc": {
                "start": 7633,
                "end": 7642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7633,
              "end": 7642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7647,
                "end": 7656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7647,
              "end": 7656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7661,
                "end": 7670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7661,
              "end": 7670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7675,
                "end": 7687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7675,
              "end": 7687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7692,
                "end": 7700
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7692,
              "end": 7700
            }
          }
        ],
        "loc": {
          "start": 7627,
          "end": 7702
        }
      },
      "loc": {
        "start": 7623,
        "end": 7702
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7733,
          "end": 7735
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7733,
        "end": 7735
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7736,
          "end": 7746
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7736,
        "end": 7746
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7747,
          "end": 7757
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7747,
        "end": 7757
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7758,
          "end": 7769
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7758,
        "end": 7769
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7770,
          "end": 7776
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7770,
        "end": 7776
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7777,
          "end": 7782
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7777,
        "end": 7782
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7783,
          "end": 7803
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7783,
        "end": 7803
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7804,
          "end": 7808
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7804,
        "end": 7808
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7809,
          "end": 7821
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7809,
        "end": 7821
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 28,
                "end": 36
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
                    "value": "translations",
                    "loc": {
                      "start": 43,
                      "end": 55
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
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 259,
                      "end": 268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 259,
                    "end": 268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
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
                          "value": "canComment",
                          "loc": {
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 553,
                        "end": 557
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 557
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 571,
                            "end": 579
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 568,
                          "end": 579
                        }
                      }
                    ],
                    "loc": {
                      "start": 558,
                      "end": 585
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 585
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 597,
                        "end": 601
                      }
                    },
                    "loc": {
                      "start": 597,
                      "end": 601
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 615,
                            "end": 623
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 612,
                          "end": 623
                        }
                      }
                    ],
                    "loc": {
                      "start": 602,
                      "end": 629
                    }
                  },
                  "loc": {
                    "start": 590,
                    "end": 629
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 631
              }
            },
            "loc": {
              "start": 534,
              "end": 631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 632,
                "end": 643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 632,
              "end": 643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 644,
                "end": 658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 644,
              "end": 658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 659,
                "end": 664
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 659,
              "end": 664
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 665,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 665,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 675,
                "end": 679
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 689,
                      "end": 697
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 686,
                    "end": 697
                  }
                }
              ],
              "loc": {
                "start": 680,
                "end": 699
              }
            },
            "loc": {
              "start": 675,
              "end": 699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 700,
                "end": 714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 700,
              "end": 714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 715,
                "end": 720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 715,
              "end": 720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 721,
                "end": 724
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
                    "value": "canDelete",
                    "loc": {
                      "start": 731,
                      "end": 740
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 731,
                    "end": 740
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 745,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 745,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 777,
                      "end": 786
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 786
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 791,
                      "end": 798
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 791,
                    "end": 798
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 803,
                      "end": 811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 803,
                    "end": 811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 816,
                      "end": 828
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 816,
                    "end": 828
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 833,
                      "end": 841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 833,
                    "end": 841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 846,
                      "end": 854
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 846,
                    "end": 854
                  }
                }
              ],
              "loc": {
                "start": 725,
                "end": 856
              }
            },
            "loc": {
              "start": 721,
              "end": 856
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 858
        }
      },
      "loc": {
        "start": 1,
        "end": 858
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 868,
          "end": 875
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 879,
            "end": 882
          }
        },
        "loc": {
          "start": 879,
          "end": 882
        }
      },
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
                "start": 885,
                "end": 887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 885,
              "end": 887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 888,
                "end": 897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 888,
              "end": 897
            }
          }
        ],
        "loc": {
          "start": 883,
          "end": 899
        }
      },
      "loc": {
        "start": 859,
        "end": 899
      }
    },
    "Code_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_list",
        "loc": {
          "start": 909,
          "end": 918
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 922,
            "end": 926
          }
        },
        "loc": {
          "start": 922,
          "end": 926
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 929,
                "end": 937
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
                    "value": "translations",
                    "loc": {
                      "start": 944,
                      "end": 956
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
                            "start": 967,
                            "end": 969
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 967,
                          "end": 969
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 978,
                            "end": 986
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 978,
                          "end": 986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 995,
                            "end": 1006
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 995,
                          "end": 1006
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 1015,
                            "end": 1027
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1015,
                          "end": 1027
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1036,
                            "end": 1040
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1036,
                          "end": 1040
                        }
                      }
                    ],
                    "loc": {
                      "start": 957,
                      "end": 1046
                    }
                  },
                  "loc": {
                    "start": 944,
                    "end": 1046
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1051,
                      "end": 1053
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1051,
                    "end": 1053
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1058,
                      "end": 1068
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1058,
                    "end": 1068
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1073,
                      "end": 1083
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1073,
                    "end": 1083
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1088,
                      "end": 1098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1088,
                    "end": 1098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1103,
                      "end": 1112
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1103,
                    "end": 1112
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1117,
                      "end": 1125
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1117,
                    "end": 1125
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1130,
                      "end": 1139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1130,
                    "end": 1139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeLanguage",
                    "loc": {
                      "start": 1144,
                      "end": 1156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1144,
                    "end": 1156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeType",
                    "loc": {
                      "start": 1161,
                      "end": 1169
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1161,
                    "end": 1169
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 1174,
                      "end": 1181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1174,
                    "end": 1181
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1186,
                      "end": 1198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1186,
                    "end": 1198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1203,
                      "end": 1215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "calledByRoutineVersionsCount",
                    "loc": {
                      "start": 1220,
                      "end": 1248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1220,
                    "end": 1248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1253,
                      "end": 1266
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1253,
                    "end": 1266
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1271,
                      "end": 1293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1271,
                    "end": 1293
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1298,
                      "end": 1308
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1298,
                    "end": 1308
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1313,
                      "end": 1325
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1313,
                    "end": 1325
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1330,
                      "end": 1333
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
                          "value": "canComment",
                          "loc": {
                            "start": 1344,
                            "end": 1354
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1344,
                          "end": 1354
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1363,
                            "end": 1370
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1363,
                          "end": 1370
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1379,
                            "end": 1388
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1379,
                          "end": 1388
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1397,
                            "end": 1406
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1397,
                          "end": 1406
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1415,
                            "end": 1424
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1415,
                          "end": 1424
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1433,
                            "end": 1439
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1433,
                          "end": 1439
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1448,
                            "end": 1455
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1448,
                          "end": 1455
                        }
                      }
                    ],
                    "loc": {
                      "start": 1334,
                      "end": 1461
                    }
                  },
                  "loc": {
                    "start": 1330,
                    "end": 1461
                  }
                }
              ],
              "loc": {
                "start": 938,
                "end": 1463
              }
            },
            "loc": {
              "start": 929,
              "end": 1463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1464,
                "end": 1466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1464,
              "end": 1466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1467,
                "end": 1477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1467,
              "end": 1477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1478,
                "end": 1488
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1478,
              "end": 1488
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1489,
                "end": 1498
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1489,
              "end": 1498
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1499,
                "end": 1510
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1499,
              "end": 1510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1511,
                "end": 1517
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1527,
                      "end": 1537
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1524,
                    "end": 1537
                  }
                }
              ],
              "loc": {
                "start": 1518,
                "end": 1539
              }
            },
            "loc": {
              "start": 1511,
              "end": 1539
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1540,
                "end": 1545
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 1559,
                        "end": 1563
                      }
                    },
                    "loc": {
                      "start": 1559,
                      "end": 1563
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 1577,
                            "end": 1585
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1574,
                          "end": 1585
                        }
                      }
                    ],
                    "loc": {
                      "start": 1564,
                      "end": 1591
                    }
                  },
                  "loc": {
                    "start": 1552,
                    "end": 1591
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1603,
                        "end": 1607
                      }
                    },
                    "loc": {
                      "start": 1603,
                      "end": 1607
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1621,
                            "end": 1629
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1618,
                          "end": 1629
                        }
                      }
                    ],
                    "loc": {
                      "start": 1608,
                      "end": 1635
                    }
                  },
                  "loc": {
                    "start": 1596,
                    "end": 1635
                  }
                }
              ],
              "loc": {
                "start": 1546,
                "end": 1637
              }
            },
            "loc": {
              "start": 1540,
              "end": 1637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1638,
                "end": 1649
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1638,
              "end": 1649
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1650,
                "end": 1664
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1650,
              "end": 1664
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1665,
                "end": 1670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1665,
              "end": 1670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1671,
                "end": 1680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1671,
              "end": 1680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1681,
                "end": 1685
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1695,
                      "end": 1703
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1692,
                    "end": 1703
                  }
                }
              ],
              "loc": {
                "start": 1686,
                "end": 1705
              }
            },
            "loc": {
              "start": 1681,
              "end": 1705
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1706,
                "end": 1720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1706,
              "end": 1720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1721,
                "end": 1726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1721,
              "end": 1726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1727,
                "end": 1730
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
                    "value": "canDelete",
                    "loc": {
                      "start": 1737,
                      "end": 1746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1737,
                    "end": 1746
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1751,
                      "end": 1762
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1751,
                    "end": 1762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1767,
                      "end": 1778
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1767,
                    "end": 1778
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1783,
                      "end": 1792
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1783,
                    "end": 1792
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1797,
                      "end": 1804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1797,
                    "end": 1804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1809,
                      "end": 1817
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1809,
                    "end": 1817
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1822,
                      "end": 1834
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1822,
                    "end": 1834
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1839,
                      "end": 1847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1839,
                    "end": 1847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1852,
                      "end": 1860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1852,
                    "end": 1860
                  }
                }
              ],
              "loc": {
                "start": 1731,
                "end": 1862
              }
            },
            "loc": {
              "start": 1727,
              "end": 1862
            }
          }
        ],
        "loc": {
          "start": 927,
          "end": 1864
        }
      },
      "loc": {
        "start": 900,
        "end": 1864
      }
    },
    "Code_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_nav",
        "loc": {
          "start": 1874,
          "end": 1882
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 1886,
            "end": 1890
          }
        },
        "loc": {
          "start": 1886,
          "end": 1890
        }
      },
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
                "start": 1893,
                "end": 1895
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1893,
              "end": 1895
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1896,
                "end": 1905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1896,
              "end": 1905
            }
          }
        ],
        "loc": {
          "start": 1891,
          "end": 1907
        }
      },
      "loc": {
        "start": 1865,
        "end": 1907
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1917,
          "end": 1927
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1931,
            "end": 1936
          }
        },
        "loc": {
          "start": 1931,
          "end": 1936
        }
      },
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
                "start": 1939,
                "end": 1941
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1939,
              "end": 1941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1942,
                "end": 1952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1942,
              "end": 1952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1953,
                "end": 1963
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1953,
              "end": 1963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1964,
                "end": 1969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1964,
              "end": 1969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1970,
                "end": 1975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1970,
              "end": 1975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1976,
                "end": 1981
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 1995,
                        "end": 1999
                      }
                    },
                    "loc": {
                      "start": 1995,
                      "end": 1999
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 2013,
                            "end": 2021
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2010,
                          "end": 2021
                        }
                      }
                    ],
                    "loc": {
                      "start": 2000,
                      "end": 2027
                    }
                  },
                  "loc": {
                    "start": 1988,
                    "end": 2027
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2039,
                        "end": 2043
                      }
                    },
                    "loc": {
                      "start": 2039,
                      "end": 2043
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2057,
                            "end": 2065
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2054,
                          "end": 2065
                        }
                      }
                    ],
                    "loc": {
                      "start": 2044,
                      "end": 2071
                    }
                  },
                  "loc": {
                    "start": 2032,
                    "end": 2071
                  }
                }
              ],
              "loc": {
                "start": 1982,
                "end": 2073
              }
            },
            "loc": {
              "start": 1976,
              "end": 2073
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2074,
                "end": 2077
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
                    "value": "canDelete",
                    "loc": {
                      "start": 2084,
                      "end": 2093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2084,
                    "end": 2093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2098,
                      "end": 2107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2098,
                    "end": 2107
                  }
                }
              ],
              "loc": {
                "start": 2078,
                "end": 2109
              }
            },
            "loc": {
              "start": 2074,
              "end": 2109
            }
          }
        ],
        "loc": {
          "start": 1937,
          "end": 2111
        }
      },
      "loc": {
        "start": 1908,
        "end": 2111
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 2121,
          "end": 2130
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2134,
            "end": 2138
          }
        },
        "loc": {
          "start": 2134,
          "end": 2138
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 2141,
                "end": 2149
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
                    "value": "translations",
                    "loc": {
                      "start": 2156,
                      "end": 2168
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
                            "start": 2179,
                            "end": 2181
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2179,
                          "end": 2181
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2190,
                            "end": 2198
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2190,
                          "end": 2198
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2207,
                            "end": 2218
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2207,
                          "end": 2218
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2227,
                            "end": 2231
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2227,
                          "end": 2231
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 2240,
                            "end": 2245
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
                                  "start": 2260,
                                  "end": 2262
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2260,
                                "end": 2262
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 2275,
                                  "end": 2284
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2275,
                                "end": 2284
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 2297,
                                  "end": 2301
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2297,
                                "end": 2301
                              }
                            }
                          ],
                          "loc": {
                            "start": 2246,
                            "end": 2311
                          }
                        },
                        "loc": {
                          "start": 2240,
                          "end": 2311
                        }
                      }
                    ],
                    "loc": {
                      "start": 2169,
                      "end": 2317
                    }
                  },
                  "loc": {
                    "start": 2156,
                    "end": 2317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2322,
                      "end": 2324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2322,
                    "end": 2324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2329,
                      "end": 2339
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2329,
                    "end": 2339
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2344,
                      "end": 2354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2344,
                    "end": 2354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 2359,
                      "end": 2367
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2359,
                    "end": 2367
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 2372,
                      "end": 2381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2372,
                    "end": 2381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2386,
                      "end": 2398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2386,
                    "end": 2398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 2403,
                      "end": 2415
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2403,
                    "end": 2415
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 2420,
                      "end": 2432
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2420,
                    "end": 2432
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2437,
                      "end": 2440
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
                          "value": "canComment",
                          "loc": {
                            "start": 2451,
                            "end": 2461
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2451,
                          "end": 2461
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 2470,
                            "end": 2477
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2470,
                          "end": 2477
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2486,
                            "end": 2495
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2486,
                          "end": 2495
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2504,
                            "end": 2513
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2504,
                          "end": 2513
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2522,
                            "end": 2531
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2522,
                          "end": 2531
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 2540,
                            "end": 2546
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2540,
                          "end": 2546
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2555,
                            "end": 2562
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2555,
                          "end": 2562
                        }
                      }
                    ],
                    "loc": {
                      "start": 2441,
                      "end": 2568
                    }
                  },
                  "loc": {
                    "start": 2437,
                    "end": 2568
                  }
                }
              ],
              "loc": {
                "start": 2150,
                "end": 2570
              }
            },
            "loc": {
              "start": 2141,
              "end": 2570
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2571,
                "end": 2573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2571,
              "end": 2573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2574,
                "end": 2584
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2574,
              "end": 2584
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2585,
                "end": 2595
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2585,
              "end": 2595
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2596,
                "end": 2605
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2596,
              "end": 2605
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2606,
                "end": 2617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2606,
              "end": 2617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2618,
                "end": 2624
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2634,
                      "end": 2644
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2631,
                    "end": 2644
                  }
                }
              ],
              "loc": {
                "start": 2625,
                "end": 2646
              }
            },
            "loc": {
              "start": 2618,
              "end": 2646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2647,
                "end": 2652
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 2666,
                        "end": 2670
                      }
                    },
                    "loc": {
                      "start": 2666,
                      "end": 2670
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 2684,
                            "end": 2692
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2681,
                          "end": 2692
                        }
                      }
                    ],
                    "loc": {
                      "start": 2671,
                      "end": 2698
                    }
                  },
                  "loc": {
                    "start": 2659,
                    "end": 2698
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2710,
                        "end": 2714
                      }
                    },
                    "loc": {
                      "start": 2710,
                      "end": 2714
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2728,
                            "end": 2736
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2725,
                          "end": 2736
                        }
                      }
                    ],
                    "loc": {
                      "start": 2715,
                      "end": 2742
                    }
                  },
                  "loc": {
                    "start": 2703,
                    "end": 2742
                  }
                }
              ],
              "loc": {
                "start": 2653,
                "end": 2744
              }
            },
            "loc": {
              "start": 2647,
              "end": 2744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2745,
                "end": 2756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2745,
              "end": 2756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2757,
                "end": 2771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2757,
              "end": 2771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2772,
                "end": 2777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2772,
              "end": 2777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2778,
                "end": 2787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2778,
              "end": 2787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2788,
                "end": 2792
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2802,
                      "end": 2810
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2799,
                    "end": 2810
                  }
                }
              ],
              "loc": {
                "start": 2793,
                "end": 2812
              }
            },
            "loc": {
              "start": 2788,
              "end": 2812
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2813,
                "end": 2827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2813,
              "end": 2827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2828,
                "end": 2833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2828,
              "end": 2833
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2834,
                "end": 2837
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
                    "value": "canDelete",
                    "loc": {
                      "start": 2844,
                      "end": 2853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2844,
                    "end": 2853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2858,
                      "end": 2869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2858,
                    "end": 2869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2874,
                      "end": 2885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2874,
                    "end": 2885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2890,
                      "end": 2899
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2890,
                    "end": 2899
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2904,
                      "end": 2911
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2904,
                    "end": 2911
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2916,
                      "end": 2924
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2916,
                    "end": 2924
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2929,
                      "end": 2941
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2929,
                    "end": 2941
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2946,
                      "end": 2954
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2946,
                    "end": 2954
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2959,
                      "end": 2967
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2959,
                    "end": 2967
                  }
                }
              ],
              "loc": {
                "start": 2838,
                "end": 2969
              }
            },
            "loc": {
              "start": 2834,
              "end": 2969
            }
          }
        ],
        "loc": {
          "start": 2139,
          "end": 2971
        }
      },
      "loc": {
        "start": 2112,
        "end": 2971
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2981,
          "end": 2989
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2993,
            "end": 2997
          }
        },
        "loc": {
          "start": 2993,
          "end": 2997
        }
      },
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
                "start": 3000,
                "end": 3002
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3000,
              "end": 3002
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3003,
                "end": 3012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3003,
              "end": 3012
            }
          }
        ],
        "loc": {
          "start": 2998,
          "end": 3014
        }
      },
      "loc": {
        "start": 2972,
        "end": 3014
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 3024,
          "end": 3036
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3040,
            "end": 3047
          }
        },
        "loc": {
          "start": 3040,
          "end": 3047
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 3050,
                "end": 3058
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
                    "value": "translations",
                    "loc": {
                      "start": 3065,
                      "end": 3077
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
                            "start": 3088,
                            "end": 3090
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3088,
                          "end": 3090
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3099,
                            "end": 3107
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3099,
                          "end": 3107
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3116,
                            "end": 3127
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3116,
                          "end": 3127
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3136,
                            "end": 3140
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3136,
                          "end": 3140
                        }
                      }
                    ],
                    "loc": {
                      "start": 3078,
                      "end": 3146
                    }
                  },
                  "loc": {
                    "start": 3065,
                    "end": 3146
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3151,
                      "end": 3153
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3151,
                    "end": 3153
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3158,
                      "end": 3168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3158,
                    "end": 3168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3173,
                      "end": 3183
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3173,
                    "end": 3183
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3188,
                      "end": 3204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3188,
                    "end": 3204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3209,
                      "end": 3217
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3209,
                    "end": 3217
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3222,
                      "end": 3231
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3222,
                    "end": 3231
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3236,
                      "end": 3248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3236,
                    "end": 3248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3253,
                      "end": 3269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3253,
                    "end": 3269
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3274,
                      "end": 3284
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3274,
                    "end": 3284
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3289,
                      "end": 3301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3289,
                    "end": 3301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3306,
                      "end": 3318
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3306,
                    "end": 3318
                  }
                }
              ],
              "loc": {
                "start": 3059,
                "end": 3320
              }
            },
            "loc": {
              "start": 3050,
              "end": 3320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3321,
                "end": 3323
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3321,
              "end": 3323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3324,
                "end": 3334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3324,
              "end": 3334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3335,
                "end": 3345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3335,
              "end": 3345
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3346,
                "end": 3355
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3346,
              "end": 3355
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3356,
                "end": 3367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3356,
              "end": 3367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3368,
                "end": 3374
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3384,
                      "end": 3394
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3381,
                    "end": 3394
                  }
                }
              ],
              "loc": {
                "start": 3375,
                "end": 3396
              }
            },
            "loc": {
              "start": 3368,
              "end": 3396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3397,
                "end": 3402
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 3416,
                        "end": 3420
                      }
                    },
                    "loc": {
                      "start": 3416,
                      "end": 3420
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 3434,
                            "end": 3442
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3431,
                          "end": 3442
                        }
                      }
                    ],
                    "loc": {
                      "start": 3421,
                      "end": 3448
                    }
                  },
                  "loc": {
                    "start": 3409,
                    "end": 3448
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3460,
                        "end": 3464
                      }
                    },
                    "loc": {
                      "start": 3460,
                      "end": 3464
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3478,
                            "end": 3486
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3475,
                          "end": 3486
                        }
                      }
                    ],
                    "loc": {
                      "start": 3465,
                      "end": 3492
                    }
                  },
                  "loc": {
                    "start": 3453,
                    "end": 3492
                  }
                }
              ],
              "loc": {
                "start": 3403,
                "end": 3494
              }
            },
            "loc": {
              "start": 3397,
              "end": 3494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3495,
                "end": 3506
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3495,
              "end": 3506
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3507,
                "end": 3521
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3507,
              "end": 3521
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3522,
                "end": 3527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3522,
              "end": 3527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3528,
                "end": 3537
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3528,
              "end": 3537
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3538,
                "end": 3542
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 3552,
                      "end": 3560
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3549,
                    "end": 3560
                  }
                }
              ],
              "loc": {
                "start": 3543,
                "end": 3562
              }
            },
            "loc": {
              "start": 3538,
              "end": 3562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3563,
                "end": 3577
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3563,
              "end": 3577
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3578,
                "end": 3583
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3578,
              "end": 3583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3584,
                "end": 3587
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
                    "value": "canDelete",
                    "loc": {
                      "start": 3594,
                      "end": 3603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3594,
                    "end": 3603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3608,
                      "end": 3619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3608,
                    "end": 3619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3624,
                      "end": 3635
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3624,
                    "end": 3635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3640,
                      "end": 3649
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3640,
                    "end": 3649
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3654,
                      "end": 3661
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3654,
                    "end": 3661
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3666,
                      "end": 3674
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3666,
                    "end": 3674
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3679,
                      "end": 3691
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3679,
                    "end": 3691
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3696,
                      "end": 3704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3696,
                    "end": 3704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3709,
                      "end": 3717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3709,
                    "end": 3717
                  }
                }
              ],
              "loc": {
                "start": 3588,
                "end": 3719
              }
            },
            "loc": {
              "start": 3584,
              "end": 3719
            }
          }
        ],
        "loc": {
          "start": 3048,
          "end": 3721
        }
      },
      "loc": {
        "start": 3015,
        "end": 3721
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3731,
          "end": 3742
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3746,
            "end": 3753
          }
        },
        "loc": {
          "start": 3746,
          "end": 3753
        }
      },
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
                "start": 3756,
                "end": 3758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3756,
              "end": 3758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3759,
                "end": 3768
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3759,
              "end": 3768
            }
          }
        ],
        "loc": {
          "start": 3754,
          "end": 3770
        }
      },
      "loc": {
        "start": 3722,
        "end": 3770
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3780,
          "end": 3793
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3797,
            "end": 3805
          }
        },
        "loc": {
          "start": 3797,
          "end": 3805
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3808,
                "end": 3820
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
                      "start": 3827,
                      "end": 3829
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3827,
                    "end": 3829
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3834,
                      "end": 3842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3834,
                    "end": 3842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3847,
                      "end": 3858
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3847,
                    "end": 3858
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3863,
                      "end": 3867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3863,
                    "end": 3867
                  }
                }
              ],
              "loc": {
                "start": 3821,
                "end": 3869
              }
            },
            "loc": {
              "start": 3808,
              "end": 3869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3870,
                "end": 3872
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3870,
              "end": 3872
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3873,
                "end": 3883
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3873,
              "end": 3883
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3884,
                "end": 3894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3884,
              "end": 3894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3895,
                "end": 3904
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
                      "start": 3911,
                      "end": 3913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3911,
                    "end": 3913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3918,
                      "end": 3928
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3918,
                    "end": 3928
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3933,
                      "end": 3943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3933,
                    "end": 3943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3948,
                      "end": 3959
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3948,
                    "end": 3959
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3964,
                      "end": 3970
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3964,
                    "end": 3970
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3975,
                      "end": 3980
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3975,
                    "end": 3980
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBotDepictingPerson",
                    "loc": {
                      "start": 3985,
                      "end": 4005
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3985,
                    "end": 4005
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4010,
                      "end": 4014
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4010,
                    "end": 4014
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 4019,
                      "end": 4031
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4019,
                    "end": 4031
                  }
                }
              ],
              "loc": {
                "start": 3905,
                "end": 4033
              }
            },
            "loc": {
              "start": 3895,
              "end": 4033
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 4034,
                "end": 4051
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4034,
              "end": 4051
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4052,
                "end": 4061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4052,
              "end": 4061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4062,
                "end": 4067
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4062,
              "end": 4067
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4068,
                "end": 4077
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4068,
              "end": 4077
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4078,
                "end": 4090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4078,
              "end": 4090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4091,
                "end": 4104
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4091,
              "end": 4104
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4105,
                "end": 4117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4105,
              "end": 4117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4118,
                "end": 4127
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 4141,
                        "end": 4144
                      }
                    },
                    "loc": {
                      "start": 4141,
                      "end": 4144
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 4158,
                            "end": 4165
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4155,
                          "end": 4165
                        }
                      }
                    ],
                    "loc": {
                      "start": 4145,
                      "end": 4171
                    }
                  },
                  "loc": {
                    "start": 4134,
                    "end": 4171
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Code",
                      "loc": {
                        "start": 4183,
                        "end": 4187
                      }
                    },
                    "loc": {
                      "start": 4183,
                      "end": 4187
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Code_nav",
                          "loc": {
                            "start": 4201,
                            "end": 4209
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4198,
                          "end": 4209
                        }
                      }
                    ],
                    "loc": {
                      "start": 4188,
                      "end": 4215
                    }
                  },
                  "loc": {
                    "start": 4176,
                    "end": 4215
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4227,
                        "end": 4231
                      }
                    },
                    "loc": {
                      "start": 4227,
                      "end": 4231
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4245,
                            "end": 4253
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4242,
                          "end": 4253
                        }
                      }
                    ],
                    "loc": {
                      "start": 4232,
                      "end": 4259
                    }
                  },
                  "loc": {
                    "start": 4220,
                    "end": 4259
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4271,
                        "end": 4278
                      }
                    },
                    "loc": {
                      "start": 4271,
                      "end": 4278
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4292,
                            "end": 4303
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4289,
                          "end": 4303
                        }
                      }
                    ],
                    "loc": {
                      "start": 4279,
                      "end": 4309
                    }
                  },
                  "loc": {
                    "start": 4264,
                    "end": 4309
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4321,
                        "end": 4328
                      }
                    },
                    "loc": {
                      "start": 4321,
                      "end": 4328
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4342,
                            "end": 4353
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4339,
                          "end": 4353
                        }
                      }
                    ],
                    "loc": {
                      "start": 4329,
                      "end": 4359
                    }
                  },
                  "loc": {
                    "start": 4314,
                    "end": 4359
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4371,
                        "end": 4379
                      }
                    },
                    "loc": {
                      "start": 4371,
                      "end": 4379
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4393,
                            "end": 4405
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4390,
                          "end": 4405
                        }
                      }
                    ],
                    "loc": {
                      "start": 4380,
                      "end": 4411
                    }
                  },
                  "loc": {
                    "start": 4364,
                    "end": 4411
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 4423,
                        "end": 4427
                      }
                    },
                    "loc": {
                      "start": 4423,
                      "end": 4427
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 4441,
                            "end": 4449
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4438,
                          "end": 4449
                        }
                      }
                    ],
                    "loc": {
                      "start": 4428,
                      "end": 4455
                    }
                  },
                  "loc": {
                    "start": 4416,
                    "end": 4455
                  }
                }
              ],
              "loc": {
                "start": 4128,
                "end": 4457
              }
            },
            "loc": {
              "start": 4118,
              "end": 4457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4458,
                "end": 4462
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 4472,
                      "end": 4480
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4469,
                    "end": 4480
                  }
                }
              ],
              "loc": {
                "start": 4463,
                "end": 4482
              }
            },
            "loc": {
              "start": 4458,
              "end": 4482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4483,
                "end": 4486
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
                    "value": "reaction",
                    "loc": {
                      "start": 4493,
                      "end": 4501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4493,
                    "end": 4501
                  }
                }
              ],
              "loc": {
                "start": 4487,
                "end": 4503
              }
            },
            "loc": {
              "start": 4483,
              "end": 4503
            }
          }
        ],
        "loc": {
          "start": 3806,
          "end": 4505
        }
      },
      "loc": {
        "start": 3771,
        "end": 4505
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4515,
          "end": 4527
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4531,
            "end": 4538
          }
        },
        "loc": {
          "start": 4531,
          "end": 4538
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4541,
                "end": 4549
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
                    "value": "translations",
                    "loc": {
                      "start": 4556,
                      "end": 4568
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
                            "start": 4579,
                            "end": 4581
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4579,
                          "end": 4581
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4590,
                            "end": 4598
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4590,
                          "end": 4598
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4607,
                            "end": 4618
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4607,
                          "end": 4618
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4627,
                            "end": 4639
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4627,
                          "end": 4639
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4648,
                            "end": 4652
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4648,
                          "end": 4652
                        }
                      }
                    ],
                    "loc": {
                      "start": 4569,
                      "end": 4658
                    }
                  },
                  "loc": {
                    "start": 4556,
                    "end": 4658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4663,
                      "end": 4665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4663,
                    "end": 4665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4670,
                      "end": 4680
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4670,
                    "end": 4680
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4685,
                      "end": 4695
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4685,
                    "end": 4695
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4700,
                      "end": 4711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4700,
                    "end": 4711
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4716,
                      "end": 4729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4716,
                    "end": 4729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4734,
                      "end": 4744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4734,
                    "end": 4744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4749,
                      "end": 4758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4749,
                    "end": 4758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4763,
                      "end": 4771
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4763,
                    "end": 4771
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4776,
                      "end": 4785
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4776,
                    "end": 4785
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 4790,
                      "end": 4801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4790,
                    "end": 4801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4806,
                      "end": 4816
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4806,
                    "end": 4816
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4821,
                      "end": 4833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4821,
                    "end": 4833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4838,
                      "end": 4852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4838,
                    "end": 4852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4857,
                      "end": 4869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4857,
                    "end": 4869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4874,
                      "end": 4886
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4874,
                    "end": 4886
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4891,
                      "end": 4904
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4891,
                    "end": 4904
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4909,
                      "end": 4931
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4909,
                    "end": 4931
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 4936,
                      "end": 4946
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4936,
                    "end": 4946
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 4951,
                      "end": 4962
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4951,
                    "end": 4962
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 4967,
                      "end": 4977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4967,
                    "end": 4977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4982,
                      "end": 4996
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4982,
                    "end": 4996
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 5001,
                      "end": 5013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5001,
                    "end": 5013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5018,
                      "end": 5030
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5018,
                    "end": 5030
                  }
                }
              ],
              "loc": {
                "start": 4550,
                "end": 5032
              }
            },
            "loc": {
              "start": 4541,
              "end": 5032
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5033,
                "end": 5035
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5033,
              "end": 5035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5036,
                "end": 5046
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5036,
              "end": 5046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5047,
                "end": 5057
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5047,
              "end": 5057
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5058,
                "end": 5068
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5058,
              "end": 5068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5069,
                "end": 5078
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5069,
              "end": 5078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5079,
                "end": 5090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5079,
              "end": 5090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5091,
                "end": 5097
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 5107,
                      "end": 5117
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5104,
                    "end": 5117
                  }
                }
              ],
              "loc": {
                "start": 5098,
                "end": 5119
              }
            },
            "loc": {
              "start": 5091,
              "end": 5119
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5120,
                "end": 5125
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 5139,
                        "end": 5143
                      }
                    },
                    "loc": {
                      "start": 5139,
                      "end": 5143
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 5157,
                            "end": 5165
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5154,
                          "end": 5165
                        }
                      }
                    ],
                    "loc": {
                      "start": 5144,
                      "end": 5171
                    }
                  },
                  "loc": {
                    "start": 5132,
                    "end": 5171
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5183,
                        "end": 5187
                      }
                    },
                    "loc": {
                      "start": 5183,
                      "end": 5187
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5201,
                            "end": 5209
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5198,
                          "end": 5209
                        }
                      }
                    ],
                    "loc": {
                      "start": 5188,
                      "end": 5215
                    }
                  },
                  "loc": {
                    "start": 5176,
                    "end": 5215
                  }
                }
              ],
              "loc": {
                "start": 5126,
                "end": 5217
              }
            },
            "loc": {
              "start": 5120,
              "end": 5217
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5218,
                "end": 5229
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5218,
              "end": 5229
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5230,
                "end": 5244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5230,
              "end": 5244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5245,
                "end": 5250
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5245,
              "end": 5250
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5251,
                "end": 5260
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5251,
              "end": 5260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5261,
                "end": 5265
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 5275,
                      "end": 5283
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5272,
                    "end": 5283
                  }
                }
              ],
              "loc": {
                "start": 5266,
                "end": 5285
              }
            },
            "loc": {
              "start": 5261,
              "end": 5285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5286,
                "end": 5300
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5286,
              "end": 5300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5301,
                "end": 5306
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5301,
              "end": 5306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5307,
                "end": 5310
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
                    "value": "canComment",
                    "loc": {
                      "start": 5317,
                      "end": 5327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5317,
                    "end": 5327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5332,
                      "end": 5341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5332,
                    "end": 5341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5346,
                      "end": 5357
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5346,
                    "end": 5357
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5362,
                      "end": 5371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5362,
                    "end": 5371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5376,
                      "end": 5383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5376,
                    "end": 5383
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5388,
                      "end": 5396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5388,
                    "end": 5396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5401,
                      "end": 5413
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5401,
                    "end": 5413
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5418,
                      "end": 5426
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5418,
                    "end": 5426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5431,
                      "end": 5439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5431,
                    "end": 5439
                  }
                }
              ],
              "loc": {
                "start": 5311,
                "end": 5441
              }
            },
            "loc": {
              "start": 5307,
              "end": 5441
            }
          }
        ],
        "loc": {
          "start": 4539,
          "end": 5443
        }
      },
      "loc": {
        "start": 4506,
        "end": 5443
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5453,
          "end": 5464
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5468,
            "end": 5475
          }
        },
        "loc": {
          "start": 5468,
          "end": 5475
        }
      },
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
                "start": 5478,
                "end": 5480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5478,
              "end": 5480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5481,
                "end": 5491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5481,
              "end": 5491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5492,
                "end": 5501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5492,
              "end": 5501
            }
          }
        ],
        "loc": {
          "start": 5476,
          "end": 5503
        }
      },
      "loc": {
        "start": 5444,
        "end": 5503
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 5513,
          "end": 5526
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 5530,
            "end": 5538
          }
        },
        "loc": {
          "start": 5530,
          "end": 5538
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5541,
                "end": 5549
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
                    "value": "translations",
                    "loc": {
                      "start": 5556,
                      "end": 5568
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
                            "start": 5579,
                            "end": 5581
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5579,
                          "end": 5581
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5590,
                            "end": 5598
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5590,
                          "end": 5598
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5607,
                            "end": 5618
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5607,
                          "end": 5618
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5627,
                            "end": 5639
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5627,
                          "end": 5639
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5648,
                            "end": 5652
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5648,
                          "end": 5652
                        }
                      }
                    ],
                    "loc": {
                      "start": 5569,
                      "end": 5658
                    }
                  },
                  "loc": {
                    "start": 5556,
                    "end": 5658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5663,
                      "end": 5665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5663,
                    "end": 5665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5670,
                      "end": 5680
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5670,
                    "end": 5680
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5685,
                      "end": 5695
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5685,
                    "end": 5695
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5700,
                      "end": 5710
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5700,
                    "end": 5710
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 5715,
                      "end": 5721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5715,
                    "end": 5721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5726,
                      "end": 5734
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5726,
                    "end": 5734
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5739,
                      "end": 5748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5739,
                    "end": 5748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5753,
                      "end": 5760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5753,
                    "end": 5760
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 5765,
                      "end": 5777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5765,
                    "end": 5777
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 5782,
                      "end": 5787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5782,
                    "end": 5787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 5792,
                      "end": 5795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5792,
                    "end": 5795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5800,
                      "end": 5812
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5800,
                    "end": 5812
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5817,
                      "end": 5829
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5817,
                    "end": 5829
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5834,
                      "end": 5847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5834,
                    "end": 5847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5852,
                      "end": 5874
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5852,
                    "end": 5874
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5879,
                      "end": 5889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5879,
                    "end": 5889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5894,
                      "end": 5906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5894,
                    "end": 5906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5911,
                      "end": 5914
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
                          "value": "canComment",
                          "loc": {
                            "start": 5925,
                            "end": 5935
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5925,
                          "end": 5935
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5944,
                            "end": 5951
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5944,
                          "end": 5951
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5960,
                            "end": 5969
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5960,
                          "end": 5969
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5978,
                            "end": 5987
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5978,
                          "end": 5987
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5996,
                            "end": 6005
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5996,
                          "end": 6005
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6014,
                            "end": 6020
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6014,
                          "end": 6020
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6029,
                            "end": 6036
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6029,
                          "end": 6036
                        }
                      }
                    ],
                    "loc": {
                      "start": 5915,
                      "end": 6042
                    }
                  },
                  "loc": {
                    "start": 5911,
                    "end": 6042
                  }
                }
              ],
              "loc": {
                "start": 5550,
                "end": 6044
              }
            },
            "loc": {
              "start": 5541,
              "end": 6044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6045,
                "end": 6047
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6045,
              "end": 6047
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6048,
                "end": 6058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6048,
              "end": 6058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6059,
                "end": 6069
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6059,
              "end": 6069
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6070,
                "end": 6079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6070,
              "end": 6079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6080,
                "end": 6091
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6080,
              "end": 6091
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6092,
                "end": 6098
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 6108,
                      "end": 6118
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6105,
                    "end": 6118
                  }
                }
              ],
              "loc": {
                "start": 6099,
                "end": 6120
              }
            },
            "loc": {
              "start": 6092,
              "end": 6120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6121,
                "end": 6126
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 6140,
                        "end": 6144
                      }
                    },
                    "loc": {
                      "start": 6140,
                      "end": 6144
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 6158,
                            "end": 6166
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6155,
                          "end": 6166
                        }
                      }
                    ],
                    "loc": {
                      "start": 6145,
                      "end": 6172
                    }
                  },
                  "loc": {
                    "start": 6133,
                    "end": 6172
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6184,
                        "end": 6188
                      }
                    },
                    "loc": {
                      "start": 6184,
                      "end": 6188
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6202,
                            "end": 6210
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6199,
                          "end": 6210
                        }
                      }
                    ],
                    "loc": {
                      "start": 6189,
                      "end": 6216
                    }
                  },
                  "loc": {
                    "start": 6177,
                    "end": 6216
                  }
                }
              ],
              "loc": {
                "start": 6127,
                "end": 6218
              }
            },
            "loc": {
              "start": 6121,
              "end": 6218
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6219,
                "end": 6230
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6219,
              "end": 6230
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6231,
                "end": 6245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6231,
              "end": 6245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6246,
                "end": 6251
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6246,
              "end": 6251
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6252,
                "end": 6261
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6252,
              "end": 6261
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6262,
                "end": 6266
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6276,
                      "end": 6284
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6273,
                    "end": 6284
                  }
                }
              ],
              "loc": {
                "start": 6267,
                "end": 6286
              }
            },
            "loc": {
              "start": 6262,
              "end": 6286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6287,
                "end": 6301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6287,
              "end": 6301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6302,
                "end": 6307
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6302,
              "end": 6307
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6308,
                "end": 6311
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
                    "value": "canDelete",
                    "loc": {
                      "start": 6318,
                      "end": 6327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6318,
                    "end": 6327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6332,
                      "end": 6343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6332,
                    "end": 6343
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6348,
                      "end": 6359
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6348,
                    "end": 6359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6364,
                      "end": 6373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6364,
                    "end": 6373
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6378,
                      "end": 6385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6378,
                    "end": 6385
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6390,
                      "end": 6398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6390,
                    "end": 6398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6403,
                      "end": 6415
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6403,
                    "end": 6415
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6420,
                      "end": 6428
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6420,
                    "end": 6428
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6433,
                      "end": 6441
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6433,
                    "end": 6441
                  }
                }
              ],
              "loc": {
                "start": 6312,
                "end": 6443
              }
            },
            "loc": {
              "start": 6308,
              "end": 6443
            }
          }
        ],
        "loc": {
          "start": 5539,
          "end": 6445
        }
      },
      "loc": {
        "start": 5504,
        "end": 6445
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 6455,
          "end": 6467
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6471,
            "end": 6479
          }
        },
        "loc": {
          "start": 6471,
          "end": 6479
        }
      },
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
                "start": 6482,
                "end": 6484
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6482,
              "end": 6484
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6485,
                "end": 6494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6485,
              "end": 6494
            }
          }
        ],
        "loc": {
          "start": 6480,
          "end": 6496
        }
      },
      "loc": {
        "start": 6446,
        "end": 6496
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6506,
          "end": 6514
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6518,
            "end": 6521
          }
        },
        "loc": {
          "start": 6518,
          "end": 6521
        }
      },
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
                "start": 6524,
                "end": 6526
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6524,
              "end": 6526
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6527,
                "end": 6537
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6527,
              "end": 6537
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6538,
                "end": 6541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6538,
              "end": 6541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6542,
                "end": 6551
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6542,
              "end": 6551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6552,
                "end": 6564
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
                      "start": 6571,
                      "end": 6573
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6571,
                    "end": 6573
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6578,
                      "end": 6586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6578,
                    "end": 6586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6591,
                      "end": 6602
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6591,
                    "end": 6602
                  }
                }
              ],
              "loc": {
                "start": 6565,
                "end": 6604
              }
            },
            "loc": {
              "start": 6552,
              "end": 6604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6605,
                "end": 6608
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
                    "value": "isOwn",
                    "loc": {
                      "start": 6615,
                      "end": 6620
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6615,
                    "end": 6620
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6625,
                      "end": 6637
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6625,
                    "end": 6637
                  }
                }
              ],
              "loc": {
                "start": 6609,
                "end": 6639
              }
            },
            "loc": {
              "start": 6605,
              "end": 6639
            }
          }
        ],
        "loc": {
          "start": 6522,
          "end": 6641
        }
      },
      "loc": {
        "start": 6497,
        "end": 6641
      }
    },
    "Team_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_list",
        "loc": {
          "start": 6651,
          "end": 6660
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 6664,
            "end": 6668
          }
        },
        "loc": {
          "start": 6664,
          "end": 6668
        }
      },
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
                "start": 6671,
                "end": 6673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6671,
              "end": 6673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 6674,
                "end": 6685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6674,
              "end": 6685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 6686,
                "end": 6692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6686,
              "end": 6692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6693,
                "end": 6703
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6693,
              "end": 6703
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6704,
                "end": 6714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6704,
              "end": 6714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 6715,
                "end": 6733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6715,
              "end": 6733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6734,
                "end": 6743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6734,
              "end": 6743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6744,
                "end": 6757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6744,
              "end": 6757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 6758,
                "end": 6770
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6758,
              "end": 6770
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 6771,
                "end": 6783
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6771,
              "end": 6783
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6784,
                "end": 6796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6784,
              "end": 6796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6797,
                "end": 6806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6797,
              "end": 6806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6807,
                "end": 6811
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6821,
                      "end": 6829
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6818,
                    "end": 6829
                  }
                }
              ],
              "loc": {
                "start": 6812,
                "end": 6831
              }
            },
            "loc": {
              "start": 6807,
              "end": 6831
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6832,
                "end": 6844
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
                      "start": 6851,
                      "end": 6853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6851,
                    "end": 6853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6858,
                      "end": 6866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6858,
                    "end": 6866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 6871,
                      "end": 6874
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6871,
                    "end": 6874
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6879,
                      "end": 6883
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6879,
                    "end": 6883
                  }
                }
              ],
              "loc": {
                "start": 6845,
                "end": 6885
              }
            },
            "loc": {
              "start": 6832,
              "end": 6885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6886,
                "end": 6889
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
                    "value": "canAddMembers",
                    "loc": {
                      "start": 6896,
                      "end": 6909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6896,
                    "end": 6909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6914,
                      "end": 6923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6914,
                    "end": 6923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6928,
                      "end": 6939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6928,
                    "end": 6939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6944,
                      "end": 6953
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6944,
                    "end": 6953
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6958,
                      "end": 6967
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6958,
                    "end": 6967
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6972,
                      "end": 6979
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6972,
                    "end": 6979
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6984,
                      "end": 6996
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6984,
                    "end": 6996
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7001,
                      "end": 7009
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7001,
                    "end": 7009
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7014,
                      "end": 7028
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
                            "start": 7039,
                            "end": 7041
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7039,
                          "end": 7041
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7050,
                            "end": 7060
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7050,
                          "end": 7060
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7069,
                            "end": 7079
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7069,
                          "end": 7079
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7088,
                            "end": 7095
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7088,
                          "end": 7095
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7104,
                            "end": 7115
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7104,
                          "end": 7115
                        }
                      }
                    ],
                    "loc": {
                      "start": 7029,
                      "end": 7121
                    }
                  },
                  "loc": {
                    "start": 7014,
                    "end": 7121
                  }
                }
              ],
              "loc": {
                "start": 6890,
                "end": 7123
              }
            },
            "loc": {
              "start": 6886,
              "end": 7123
            }
          }
        ],
        "loc": {
          "start": 6669,
          "end": 7125
        }
      },
      "loc": {
        "start": 6642,
        "end": 7125
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 7135,
          "end": 7143
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 7147,
            "end": 7151
          }
        },
        "loc": {
          "start": 7147,
          "end": 7151
        }
      },
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
                "start": 7154,
                "end": 7156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7154,
              "end": 7156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7157,
                "end": 7168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7157,
              "end": 7168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7169,
                "end": 7175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7169,
              "end": 7175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7176,
                "end": 7188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7176,
              "end": 7188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7189,
                "end": 7192
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
                    "value": "canAddMembers",
                    "loc": {
                      "start": 7199,
                      "end": 7212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7199,
                    "end": 7212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7217,
                      "end": 7226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7217,
                    "end": 7226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7231,
                      "end": 7242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7231,
                    "end": 7242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7247,
                      "end": 7256
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7247,
                    "end": 7256
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7261,
                      "end": 7270
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7261,
                    "end": 7270
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7275,
                      "end": 7282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7275,
                    "end": 7282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7287,
                      "end": 7299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7287,
                    "end": 7299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7304,
                      "end": 7312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7304,
                    "end": 7312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7317,
                      "end": 7331
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
                            "start": 7342,
                            "end": 7344
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7342,
                          "end": 7344
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7353,
                            "end": 7363
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7353,
                          "end": 7363
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7372,
                            "end": 7382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7372,
                          "end": 7382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7391,
                            "end": 7398
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7391,
                          "end": 7398
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7407,
                            "end": 7418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7407,
                          "end": 7418
                        }
                      }
                    ],
                    "loc": {
                      "start": 7332,
                      "end": 7424
                    }
                  },
                  "loc": {
                    "start": 7317,
                    "end": 7424
                  }
                }
              ],
              "loc": {
                "start": 7193,
                "end": 7426
              }
            },
            "loc": {
              "start": 7189,
              "end": 7426
            }
          }
        ],
        "loc": {
          "start": 7152,
          "end": 7428
        }
      },
      "loc": {
        "start": 7126,
        "end": 7428
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7438,
          "end": 7447
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7451,
            "end": 7455
          }
        },
        "loc": {
          "start": 7451,
          "end": 7455
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7458,
                "end": 7470
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
                      "start": 7477,
                      "end": 7479
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7477,
                    "end": 7479
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7484,
                      "end": 7492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7484,
                    "end": 7492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7497,
                      "end": 7500
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7497,
                    "end": 7500
                  }
                }
              ],
              "loc": {
                "start": 7471,
                "end": 7502
              }
            },
            "loc": {
              "start": 7458,
              "end": 7502
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7503,
                "end": 7505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7503,
              "end": 7505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7506,
                "end": 7516
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7506,
              "end": 7516
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7517,
                "end": 7527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7517,
              "end": 7527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7528,
                "end": 7539
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7528,
              "end": 7539
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7540,
                "end": 7546
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7540,
              "end": 7546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7547,
                "end": 7552
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7547,
              "end": 7552
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7553,
                "end": 7573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7553,
              "end": 7573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7574,
                "end": 7578
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7574,
              "end": 7578
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7579,
                "end": 7591
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7579,
              "end": 7591
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7592,
                "end": 7601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7592,
              "end": 7601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7602,
                "end": 7622
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7602,
              "end": 7622
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7623,
                "end": 7626
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
                    "value": "canDelete",
                    "loc": {
                      "start": 7633,
                      "end": 7642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7633,
                    "end": 7642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7647,
                      "end": 7656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7647,
                    "end": 7656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7661,
                      "end": 7670
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7661,
                    "end": 7670
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7675,
                      "end": 7687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7675,
                    "end": 7687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7692,
                      "end": 7700
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7692,
                    "end": 7700
                  }
                }
              ],
              "loc": {
                "start": 7627,
                "end": 7702
              }
            },
            "loc": {
              "start": 7623,
              "end": 7702
            }
          }
        ],
        "loc": {
          "start": 7456,
          "end": 7704
        }
      },
      "loc": {
        "start": 7429,
        "end": 7704
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7714,
          "end": 7722
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7726,
            "end": 7730
          }
        },
        "loc": {
          "start": 7726,
          "end": 7730
        }
      },
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
                "start": 7733,
                "end": 7735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7733,
              "end": 7735
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7736,
                "end": 7746
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7736,
              "end": 7746
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7747,
                "end": 7757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7747,
              "end": 7757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7758,
                "end": 7769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7758,
              "end": 7769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7770,
                "end": 7776
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7770,
              "end": 7776
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7777,
                "end": 7782
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7777,
              "end": 7782
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7783,
                "end": 7803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7783,
              "end": 7803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7804,
                "end": 7808
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7804,
              "end": 7808
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7809,
                "end": 7821
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7809,
              "end": 7821
            }
          }
        ],
        "loc": {
          "start": 7731,
          "end": 7823
        }
      },
      "loc": {
        "start": 7705,
        "end": 7823
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "populars",
      "loc": {
        "start": 7831,
        "end": 7839
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
              "start": 7841,
              "end": 7846
            }
          },
          "loc": {
            "start": 7840,
            "end": 7846
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularSearchInput",
              "loc": {
                "start": 7848,
                "end": 7866
              }
            },
            "loc": {
              "start": 7848,
              "end": 7866
            }
          },
          "loc": {
            "start": 7848,
            "end": 7867
          }
        },
        "directives": [],
        "loc": {
          "start": 7840,
          "end": 7867
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
            "value": "populars",
            "loc": {
              "start": 7873,
              "end": 7881
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7882,
                  "end": 7887
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7890,
                    "end": 7895
                  }
                },
                "loc": {
                  "start": 7889,
                  "end": 7895
                }
              },
              "loc": {
                "start": 7882,
                "end": 7895
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
                    "start": 7903,
                    "end": 7908
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
                          "start": 7919,
                          "end": 7925
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 7919,
                        "end": 7925
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 7934,
                          "end": 7938
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Api",
                                "loc": {
                                  "start": 7960,
                                  "end": 7963
                                }
                              },
                              "loc": {
                                "start": 7960,
                                "end": 7963
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Api_list",
                                    "loc": {
                                      "start": 7985,
                                      "end": 7993
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 7982,
                                    "end": 7993
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7964,
                                "end": 8007
                              }
                            },
                            "loc": {
                              "start": 7953,
                              "end": 8007
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Code",
                                "loc": {
                                  "start": 8027,
                                  "end": 8031
                                }
                              },
                              "loc": {
                                "start": 8027,
                                "end": 8031
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Code_list",
                                    "loc": {
                                      "start": 8053,
                                      "end": 8062
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8050,
                                    "end": 8062
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8032,
                                "end": 8076
                              }
                            },
                            "loc": {
                              "start": 8020,
                              "end": 8076
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Note",
                                "loc": {
                                  "start": 8096,
                                  "end": 8100
                                }
                              },
                              "loc": {
                                "start": 8096,
                                "end": 8100
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Note_list",
                                    "loc": {
                                      "start": 8122,
                                      "end": 8131
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8119,
                                    "end": 8131
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8101,
                                "end": 8145
                              }
                            },
                            "loc": {
                              "start": 8089,
                              "end": 8145
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 8165,
                                  "end": 8172
                                }
                              },
                              "loc": {
                                "start": 8165,
                                "end": 8172
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 8194,
                                      "end": 8206
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8191,
                                    "end": 8206
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8173,
                                "end": 8220
                              }
                            },
                            "loc": {
                              "start": 8158,
                              "end": 8220
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Question",
                                "loc": {
                                  "start": 8240,
                                  "end": 8248
                                }
                              },
                              "loc": {
                                "start": 8240,
                                "end": 8248
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Question_list",
                                    "loc": {
                                      "start": 8270,
                                      "end": 8283
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8267,
                                    "end": 8283
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8249,
                                "end": 8297
                              }
                            },
                            "loc": {
                              "start": 8233,
                              "end": 8297
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 8317,
                                  "end": 8324
                                }
                              },
                              "loc": {
                                "start": 8317,
                                "end": 8324
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 8346,
                                      "end": 8358
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8343,
                                    "end": 8358
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8325,
                                "end": 8372
                              }
                            },
                            "loc": {
                              "start": 8310,
                              "end": 8372
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Standard",
                                "loc": {
                                  "start": 8392,
                                  "end": 8400
                                }
                              },
                              "loc": {
                                "start": 8392,
                                "end": 8400
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Standard_list",
                                    "loc": {
                                      "start": 8422,
                                      "end": 8435
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8419,
                                    "end": 8435
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8401,
                                "end": 8449
                              }
                            },
                            "loc": {
                              "start": 8385,
                              "end": 8449
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Team",
                                "loc": {
                                  "start": 8469,
                                  "end": 8473
                                }
                              },
                              "loc": {
                                "start": 8469,
                                "end": 8473
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Team_list",
                                    "loc": {
                                      "start": 8495,
                                      "end": 8504
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8492,
                                    "end": 8504
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8474,
                                "end": 8518
                              }
                            },
                            "loc": {
                              "start": 8462,
                              "end": 8518
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "User",
                                "loc": {
                                  "start": 8538,
                                  "end": 8542
                                }
                              },
                              "loc": {
                                "start": 8538,
                                "end": 8542
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "User_list",
                                    "loc": {
                                      "start": 8564,
                                      "end": 8573
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8561,
                                    "end": 8573
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8543,
                                "end": 8587
                              }
                            },
                            "loc": {
                              "start": 8531,
                              "end": 8587
                            }
                          }
                        ],
                        "loc": {
                          "start": 7939,
                          "end": 8597
                        }
                      },
                      "loc": {
                        "start": 7934,
                        "end": 8597
                      }
                    }
                  ],
                  "loc": {
                    "start": 7909,
                    "end": 8603
                  }
                },
                "loc": {
                  "start": 7903,
                  "end": 8603
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8608,
                    "end": 8616
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
                        "value": "hasNextPage",
                        "loc": {
                          "start": 8627,
                          "end": 8638
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8627,
                        "end": 8638
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8647,
                          "end": 8659
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8647,
                        "end": 8659
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorCode",
                        "loc": {
                          "start": 8668,
                          "end": 8681
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8668,
                        "end": 8681
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8690,
                          "end": 8703
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8690,
                        "end": 8703
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8712,
                          "end": 8728
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8712,
                        "end": 8728
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8737,
                          "end": 8754
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8737,
                        "end": 8754
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8763,
                          "end": 8779
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8763,
                        "end": 8779
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 8788,
                          "end": 8805
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8788,
                        "end": 8805
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorTeam",
                        "loc": {
                          "start": 8814,
                          "end": 8827
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8814,
                        "end": 8827
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 8836,
                          "end": 8849
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8836,
                        "end": 8849
                      }
                    }
                  ],
                  "loc": {
                    "start": 8617,
                    "end": 8855
                  }
                },
                "loc": {
                  "start": 8608,
                  "end": 8855
                }
              }
            ],
            "loc": {
              "start": 7897,
              "end": 8859
            }
          },
          "loc": {
            "start": 7873,
            "end": 8859
          }
        }
      ],
      "loc": {
        "start": 7869,
        "end": 8861
      }
    },
    "loc": {
      "start": 7825,
      "end": 8861
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
