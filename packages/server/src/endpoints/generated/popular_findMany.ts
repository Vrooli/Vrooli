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
        "value": "id",
        "loc": {
          "start": 28,
          "end": 30
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 28,
        "end": 30
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 31,
          "end": 41
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 31,
        "end": 41
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 42,
          "end": 52
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 42,
        "end": 52
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 53,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 53,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 63,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 74
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 75,
          "end": 81
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
                "start": 91,
                "end": 101
              }
            },
            "directives": [],
            "loc": {
              "start": 88,
              "end": 101
            }
          }
        ],
        "loc": {
          "start": 82,
          "end": 103
        }
      },
      "loc": {
        "start": 75,
        "end": 103
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 104,
          "end": 109
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
                  "start": 123,
                  "end": 127
                }
              },
              "loc": {
                "start": 123,
                "end": 127
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
                      "start": 141,
                      "end": 149
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 138,
                    "end": 149
                  }
                }
              ],
              "loc": {
                "start": 128,
                "end": 155
              }
            },
            "loc": {
              "start": 116,
              "end": 155
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
                  "start": 167,
                  "end": 171
                }
              },
              "loc": {
                "start": 167,
                "end": 171
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
                      "start": 185,
                      "end": 193
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 182,
                    "end": 193
                  }
                }
              ],
              "loc": {
                "start": 172,
                "end": 199
              }
            },
            "loc": {
              "start": 160,
              "end": 199
            }
          }
        ],
        "loc": {
          "start": 110,
          "end": 201
        }
      },
      "loc": {
        "start": 104,
        "end": 201
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 202,
          "end": 213
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 202,
        "end": 213
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 214,
          "end": 228
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 214,
        "end": 228
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 229,
          "end": 234
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 229,
        "end": 234
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 235,
          "end": 244
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 235,
        "end": 244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 245,
          "end": 249
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
                "start": 259,
                "end": 267
              }
            },
            "directives": [],
            "loc": {
              "start": 256,
              "end": 267
            }
          }
        ],
        "loc": {
          "start": 250,
          "end": 269
        }
      },
      "loc": {
        "start": 245,
        "end": 269
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 270,
          "end": 284
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 270,
        "end": 284
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 285,
          "end": 290
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 285,
        "end": 290
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 291,
          "end": 294
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
                "start": 301,
                "end": 310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 301,
              "end": 310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 315,
                "end": 326
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 315,
              "end": 326
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 331,
                "end": 342
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 331,
              "end": 342
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 347,
                "end": 356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 347,
              "end": 356
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 361,
                "end": 368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 361,
              "end": 368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 373,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 373,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 386,
                "end": 398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 386,
              "end": 398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 403,
                "end": 411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 403,
              "end": 411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 416,
                "end": 424
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 416,
              "end": 424
            }
          }
        ],
        "loc": {
          "start": 295,
          "end": 426
        }
      },
      "loc": {
        "start": 291,
        "end": 426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 427,
          "end": 435
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
                "start": 442,
                "end": 444
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 442,
              "end": 444
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
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
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 464,
                "end": 474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 464,
              "end": 474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 479,
                "end": 487
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 479,
              "end": 487
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 492,
                "end": 505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 492,
              "end": 505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 510,
                "end": 527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 510,
              "end": 527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 532,
                "end": 542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 532,
              "end": 542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 547,
                "end": 555
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 547,
              "end": 555
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 560,
                "end": 569
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 560,
              "end": 569
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 574,
                "end": 586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 574,
              "end": 586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 591,
                "end": 603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 591,
              "end": 603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 608,
                "end": 620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 608,
              "end": 620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 625,
                "end": 628
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
                      "start": 639,
                      "end": 649
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 639,
                    "end": 649
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 658,
                      "end": 665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 658,
                    "end": 665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 674,
                      "end": 683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 674,
                    "end": 683
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 692,
                      "end": 701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 692,
                    "end": 701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 710,
                      "end": 719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 710,
                    "end": 719
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 728,
                      "end": 734
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 728,
                    "end": 734
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 743,
                      "end": 750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 743,
                    "end": 750
                  }
                }
              ],
              "loc": {
                "start": 629,
                "end": 756
              }
            },
            "loc": {
              "start": 625,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 761,
                "end": 773
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
                      "start": 784,
                      "end": 786
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 784,
                    "end": 786
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 795,
                      "end": 803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 795,
                    "end": 803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 812,
                      "end": 819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 812,
                    "end": 819
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 828,
                      "end": 832
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 828,
                    "end": 832
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 841,
                      "end": 848
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 841,
                    "end": 848
                  }
                }
              ],
              "loc": {
                "start": 774,
                "end": 854
              }
            },
            "loc": {
              "start": 761,
              "end": 854
            }
          }
        ],
        "loc": {
          "start": 436,
          "end": 856
        }
      },
      "loc": {
        "start": 427,
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
        "value": "id",
        "loc": {
          "start": 929,
          "end": 931
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 929,
        "end": 931
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 932,
          "end": 942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 932,
        "end": 942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 943,
          "end": 953
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 943,
        "end": 953
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 954,
          "end": 963
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 954,
        "end": 963
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 964,
          "end": 975
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 964,
        "end": 975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 976,
          "end": 982
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
                "start": 992,
                "end": 1002
              }
            },
            "directives": [],
            "loc": {
              "start": 989,
              "end": 1002
            }
          }
        ],
        "loc": {
          "start": 983,
          "end": 1004
        }
      },
      "loc": {
        "start": 976,
        "end": 1004
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1005,
          "end": 1010
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
                  "start": 1024,
                  "end": 1028
                }
              },
              "loc": {
                "start": 1024,
                "end": 1028
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
                      "start": 1042,
                      "end": 1050
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1039,
                    "end": 1050
                  }
                }
              ],
              "loc": {
                "start": 1029,
                "end": 1056
              }
            },
            "loc": {
              "start": 1017,
              "end": 1056
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
                  "start": 1068,
                  "end": 1072
                }
              },
              "loc": {
                "start": 1068,
                "end": 1072
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
                      "start": 1086,
                      "end": 1094
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1083,
                    "end": 1094
                  }
                }
              ],
              "loc": {
                "start": 1073,
                "end": 1100
              }
            },
            "loc": {
              "start": 1061,
              "end": 1100
            }
          }
        ],
        "loc": {
          "start": 1011,
          "end": 1102
        }
      },
      "loc": {
        "start": 1005,
        "end": 1102
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1103,
          "end": 1114
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1103,
        "end": 1114
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1115,
          "end": 1129
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1115,
        "end": 1129
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1130,
          "end": 1135
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1130,
        "end": 1135
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1136,
          "end": 1145
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1136,
        "end": 1145
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1146,
          "end": 1150
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
                "start": 1160,
                "end": 1168
              }
            },
            "directives": [],
            "loc": {
              "start": 1157,
              "end": 1168
            }
          }
        ],
        "loc": {
          "start": 1151,
          "end": 1170
        }
      },
      "loc": {
        "start": 1146,
        "end": 1170
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1171,
          "end": 1185
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1171,
        "end": 1185
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1186,
          "end": 1191
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1186,
        "end": 1191
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1192,
          "end": 1195
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
                "start": 1202,
                "end": 1211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1202,
              "end": 1211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1216,
                "end": 1227
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1216,
              "end": 1227
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1232,
                "end": 1243
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1232,
              "end": 1243
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1248,
                "end": 1257
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1248,
              "end": 1257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1262,
                "end": 1269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1262,
              "end": 1269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1274,
                "end": 1282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1274,
              "end": 1282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1287,
                "end": 1299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1287,
              "end": 1299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1304,
                "end": 1312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1304,
              "end": 1312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1317,
                "end": 1325
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1317,
              "end": 1325
            }
          }
        ],
        "loc": {
          "start": 1196,
          "end": 1327
        }
      },
      "loc": {
        "start": 1192,
        "end": 1327
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1328,
          "end": 1336
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
                "start": 1343,
                "end": 1345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1343,
              "end": 1345
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1350,
                "end": 1360
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1350,
              "end": 1360
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1365,
                "end": 1375
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1365,
              "end": 1375
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1380,
                "end": 1390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1380,
              "end": 1390
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1395,
                "end": 1404
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1395,
              "end": 1404
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1409,
                "end": 1417
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1409,
              "end": 1417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1422,
                "end": 1431
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1422,
              "end": 1431
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeLanguage",
              "loc": {
                "start": 1436,
                "end": 1448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1436,
              "end": 1448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeType",
              "loc": {
                "start": 1453,
                "end": 1461
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1453,
              "end": 1461
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 1466,
                "end": 1473
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1466,
              "end": 1473
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1478,
                "end": 1490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1478,
              "end": 1490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1495,
                "end": 1507
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1495,
              "end": 1507
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "calledByRoutineVersionsCount",
              "loc": {
                "start": 1512,
                "end": 1540
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1512,
              "end": 1540
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1545,
                "end": 1558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1545,
              "end": 1558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1563,
                "end": 1585
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1563,
              "end": 1585
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1590,
                "end": 1600
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1590,
              "end": 1600
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1605,
                "end": 1617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1605,
              "end": 1617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1622,
                "end": 1625
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
                      "start": 1636,
                      "end": 1646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1636,
                    "end": 1646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1655,
                      "end": 1662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1655,
                    "end": 1662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
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
                    "value": "canReport",
                    "loc": {
                      "start": 1689,
                      "end": 1698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1689,
                    "end": 1698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1707,
                      "end": 1716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1707,
                    "end": 1716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1725,
                      "end": 1731
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1725,
                    "end": 1731
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1740,
                      "end": 1747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1740,
                    "end": 1747
                  }
                }
              ],
              "loc": {
                "start": 1626,
                "end": 1753
              }
            },
            "loc": {
              "start": 1622,
              "end": 1753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1758,
                "end": 1770
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
                      "start": 1781,
                      "end": 1783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1781,
                    "end": 1783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1792,
                      "end": 1800
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1792,
                    "end": 1800
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1809,
                      "end": 1820
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1809,
                    "end": 1820
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 1829,
                      "end": 1841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1829,
                    "end": 1841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1850,
                      "end": 1854
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1850,
                    "end": 1854
                  }
                }
              ],
              "loc": {
                "start": 1771,
                "end": 1860
              }
            },
            "loc": {
              "start": 1758,
              "end": 1860
            }
          }
        ],
        "loc": {
          "start": 1337,
          "end": 1862
        }
      },
      "loc": {
        "start": 1328,
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
        "value": "id",
        "loc": {
          "start": 2141,
          "end": 2143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2141,
        "end": 2143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2144,
          "end": 2154
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2144,
        "end": 2154
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2155,
          "end": 2165
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2155,
        "end": 2165
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2166,
          "end": 2175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2166,
        "end": 2175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 2176,
          "end": 2187
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2176,
        "end": 2187
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2188,
          "end": 2194
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
                "start": 2204,
                "end": 2214
              }
            },
            "directives": [],
            "loc": {
              "start": 2201,
              "end": 2214
            }
          }
        ],
        "loc": {
          "start": 2195,
          "end": 2216
        }
      },
      "loc": {
        "start": 2188,
        "end": 2216
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 2217,
          "end": 2222
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
                  "start": 2236,
                  "end": 2240
                }
              },
              "loc": {
                "start": 2236,
                "end": 2240
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
                      "start": 2254,
                      "end": 2262
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2251,
                    "end": 2262
                  }
                }
              ],
              "loc": {
                "start": 2241,
                "end": 2268
              }
            },
            "loc": {
              "start": 2229,
              "end": 2268
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
                  "start": 2280,
                  "end": 2284
                }
              },
              "loc": {
                "start": 2280,
                "end": 2284
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
                      "start": 2298,
                      "end": 2306
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2295,
                    "end": 2306
                  }
                }
              ],
              "loc": {
                "start": 2285,
                "end": 2312
              }
            },
            "loc": {
              "start": 2273,
              "end": 2312
            }
          }
        ],
        "loc": {
          "start": 2223,
          "end": 2314
        }
      },
      "loc": {
        "start": 2217,
        "end": 2314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2315,
          "end": 2326
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2315,
        "end": 2326
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2327,
          "end": 2341
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2327,
        "end": 2341
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2342,
          "end": 2347
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2342,
        "end": 2347
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2348,
          "end": 2357
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2348,
        "end": 2357
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2358,
          "end": 2362
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
                "start": 2372,
                "end": 2380
              }
            },
            "directives": [],
            "loc": {
              "start": 2369,
              "end": 2380
            }
          }
        ],
        "loc": {
          "start": 2363,
          "end": 2382
        }
      },
      "loc": {
        "start": 2358,
        "end": 2382
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2383,
          "end": 2397
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2383,
        "end": 2397
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2398,
          "end": 2403
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2398,
        "end": 2403
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2404,
          "end": 2407
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
                "start": 2414,
                "end": 2423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2414,
              "end": 2423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2428,
                "end": 2439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2428,
              "end": 2439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2444,
                "end": 2455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2444,
              "end": 2455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2460,
                "end": 2469
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2460,
              "end": 2469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2474,
                "end": 2481
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2474,
              "end": 2481
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2486,
                "end": 2494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2486,
              "end": 2494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2499,
                "end": 2511
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2499,
              "end": 2511
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2516,
                "end": 2524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2516,
              "end": 2524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2529,
                "end": 2537
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2529,
              "end": 2537
            }
          }
        ],
        "loc": {
          "start": 2408,
          "end": 2539
        }
      },
      "loc": {
        "start": 2404,
        "end": 2539
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2540,
          "end": 2548
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
                "start": 2555,
                "end": 2557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2555,
              "end": 2557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2562,
                "end": 2572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2562,
              "end": 2572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2577,
                "end": 2587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2577,
              "end": 2587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 2592,
                "end": 2600
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2592,
              "end": 2600
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2605,
                "end": 2614
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2605,
              "end": 2614
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2619,
                "end": 2631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2619,
              "end": 2631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 2636,
                "end": 2648
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2636,
              "end": 2648
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 2653,
                "end": 2665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2653,
              "end": 2665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2670,
                "end": 2673
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
                      "start": 2684,
                      "end": 2694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2684,
                    "end": 2694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 2703,
                      "end": 2710
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2703,
                    "end": 2710
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2719,
                      "end": 2728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2719,
                    "end": 2728
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2737,
                      "end": 2746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2737,
                    "end": 2746
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2755,
                      "end": 2764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2755,
                    "end": 2764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 2773,
                      "end": 2779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2773,
                    "end": 2779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2788,
                      "end": 2795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2788,
                    "end": 2795
                  }
                }
              ],
              "loc": {
                "start": 2674,
                "end": 2801
              }
            },
            "loc": {
              "start": 2670,
              "end": 2801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2806,
                "end": 2818
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
                      "start": 2829,
                      "end": 2831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2829,
                    "end": 2831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2840,
                      "end": 2848
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2840,
                    "end": 2848
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2857,
                      "end": 2868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2857,
                    "end": 2868
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2877,
                      "end": 2881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2877,
                    "end": 2881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 2890,
                      "end": 2895
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
                            "start": 2910,
                            "end": 2912
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2910,
                          "end": 2912
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 2925,
                            "end": 2934
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2925,
                          "end": 2934
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 2947,
                            "end": 2951
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2947,
                          "end": 2951
                        }
                      }
                    ],
                    "loc": {
                      "start": 2896,
                      "end": 2961
                    }
                  },
                  "loc": {
                    "start": 2890,
                    "end": 2961
                  }
                }
              ],
              "loc": {
                "start": 2819,
                "end": 2967
              }
            },
            "loc": {
              "start": 2806,
              "end": 2967
            }
          }
        ],
        "loc": {
          "start": 2549,
          "end": 2969
        }
      },
      "loc": {
        "start": 2540,
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
        "value": "id",
        "loc": {
          "start": 3050,
          "end": 3052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3050,
        "end": 3052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3053,
          "end": 3063
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3053,
        "end": 3063
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3064,
          "end": 3074
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3064,
        "end": 3074
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3075,
          "end": 3084
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3075,
        "end": 3084
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3085,
          "end": 3096
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3085,
        "end": 3096
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3097,
          "end": 3103
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
                "start": 3113,
                "end": 3123
              }
            },
            "directives": [],
            "loc": {
              "start": 3110,
              "end": 3123
            }
          }
        ],
        "loc": {
          "start": 3104,
          "end": 3125
        }
      },
      "loc": {
        "start": 3097,
        "end": 3125
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3126,
          "end": 3131
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
                  "start": 3145,
                  "end": 3149
                }
              },
              "loc": {
                "start": 3145,
                "end": 3149
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
                      "start": 3163,
                      "end": 3171
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3160,
                    "end": 3171
                  }
                }
              ],
              "loc": {
                "start": 3150,
                "end": 3177
              }
            },
            "loc": {
              "start": 3138,
              "end": 3177
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
                  "start": 3189,
                  "end": 3193
                }
              },
              "loc": {
                "start": 3189,
                "end": 3193
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
                      "start": 3207,
                      "end": 3215
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3204,
                    "end": 3215
                  }
                }
              ],
              "loc": {
                "start": 3194,
                "end": 3221
              }
            },
            "loc": {
              "start": 3182,
              "end": 3221
            }
          }
        ],
        "loc": {
          "start": 3132,
          "end": 3223
        }
      },
      "loc": {
        "start": 3126,
        "end": 3223
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3224,
          "end": 3235
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3224,
        "end": 3235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3236,
          "end": 3250
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3236,
        "end": 3250
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3251,
          "end": 3256
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3251,
        "end": 3256
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3257,
          "end": 3266
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3257,
        "end": 3266
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3267,
          "end": 3271
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
                "start": 3281,
                "end": 3289
              }
            },
            "directives": [],
            "loc": {
              "start": 3278,
              "end": 3289
            }
          }
        ],
        "loc": {
          "start": 3272,
          "end": 3291
        }
      },
      "loc": {
        "start": 3267,
        "end": 3291
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3292,
          "end": 3306
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3292,
        "end": 3306
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3307,
          "end": 3312
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3307,
        "end": 3312
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3313,
          "end": 3316
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
                "start": 3323,
                "end": 3332
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3323,
              "end": 3332
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3337,
                "end": 3348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3337,
              "end": 3348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3353,
                "end": 3364
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3353,
              "end": 3364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3369,
                "end": 3378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3369,
              "end": 3378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3383,
                "end": 3390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3383,
              "end": 3390
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3395,
                "end": 3403
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3395,
              "end": 3403
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3408,
                "end": 3420
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3408,
              "end": 3420
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3425,
                "end": 3433
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3425,
              "end": 3433
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3438,
                "end": 3446
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3438,
              "end": 3446
            }
          }
        ],
        "loc": {
          "start": 3317,
          "end": 3448
        }
      },
      "loc": {
        "start": 3313,
        "end": 3448
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3449,
          "end": 3457
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
                "start": 3464,
                "end": 3466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3464,
              "end": 3466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3471,
                "end": 3481
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3471,
              "end": 3481
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3486,
                "end": 3496
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3486,
              "end": 3496
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3501,
                "end": 3517
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3501,
              "end": 3517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3522,
                "end": 3530
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3522,
              "end": 3530
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3535,
                "end": 3544
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3535,
              "end": 3544
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3549,
                "end": 3561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3549,
              "end": 3561
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3566,
                "end": 3582
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3566,
              "end": 3582
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3587,
                "end": 3597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3587,
              "end": 3597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3602,
                "end": 3614
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3602,
              "end": 3614
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3619,
                "end": 3631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3619,
              "end": 3631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3636,
                "end": 3648
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
                      "start": 3659,
                      "end": 3661
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3659,
                    "end": 3661
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3670,
                      "end": 3678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3670,
                    "end": 3678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3687,
                      "end": 3698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3687,
                    "end": 3698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3707,
                      "end": 3711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3707,
                    "end": 3711
                  }
                }
              ],
              "loc": {
                "start": 3649,
                "end": 3717
              }
            },
            "loc": {
              "start": 3636,
              "end": 3717
            }
          }
        ],
        "loc": {
          "start": 3458,
          "end": 3719
        }
      },
      "loc": {
        "start": 3449,
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
        "value": "id",
        "loc": {
          "start": 3808,
          "end": 3810
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3808,
        "end": 3810
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3811,
          "end": 3821
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3811,
        "end": 3821
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3822,
          "end": 3832
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3822,
        "end": 3832
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3833,
          "end": 3842
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
                "start": 3849,
                "end": 3851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3849,
              "end": 3851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3856,
                "end": 3866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3856,
              "end": 3866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3871,
                "end": 3881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3871,
              "end": 3881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3886,
                "end": 3897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3886,
              "end": 3897
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3902,
                "end": 3908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3902,
              "end": 3908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3913,
                "end": 3918
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3913,
              "end": 3918
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 3923,
                "end": 3943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3923,
              "end": 3943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3948,
                "end": 3952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3948,
              "end": 3952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3957,
                "end": 3969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3957,
              "end": 3969
            }
          }
        ],
        "loc": {
          "start": 3843,
          "end": 3971
        }
      },
      "loc": {
        "start": 3833,
        "end": 3971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 3972,
          "end": 3989
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3972,
        "end": 3989
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3990,
          "end": 3999
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3990,
        "end": 3999
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4000,
          "end": 4005
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4000,
        "end": 4005
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4006,
          "end": 4015
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4006,
        "end": 4015
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4016,
          "end": 4028
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4016,
        "end": 4028
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4029,
          "end": 4042
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4029,
        "end": 4042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4043,
          "end": 4055
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4043,
        "end": 4055
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4056,
          "end": 4065
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
                  "start": 4079,
                  "end": 4082
                }
              },
              "loc": {
                "start": 4079,
                "end": 4082
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
                      "start": 4096,
                      "end": 4103
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4093,
                    "end": 4103
                  }
                }
              ],
              "loc": {
                "start": 4083,
                "end": 4109
              }
            },
            "loc": {
              "start": 4072,
              "end": 4109
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
                  "start": 4121,
                  "end": 4125
                }
              },
              "loc": {
                "start": 4121,
                "end": 4125
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
                      "start": 4139,
                      "end": 4147
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4136,
                    "end": 4147
                  }
                }
              ],
              "loc": {
                "start": 4126,
                "end": 4153
              }
            },
            "loc": {
              "start": 4114,
              "end": 4153
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
                  "start": 4165,
                  "end": 4169
                }
              },
              "loc": {
                "start": 4165,
                "end": 4169
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
                      "start": 4183,
                      "end": 4191
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4180,
                    "end": 4191
                  }
                }
              ],
              "loc": {
                "start": 4170,
                "end": 4197
              }
            },
            "loc": {
              "start": 4158,
              "end": 4197
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
                  "start": 4209,
                  "end": 4216
                }
              },
              "loc": {
                "start": 4209,
                "end": 4216
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
                      "start": 4230,
                      "end": 4241
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4227,
                    "end": 4241
                  }
                }
              ],
              "loc": {
                "start": 4217,
                "end": 4247
              }
            },
            "loc": {
              "start": 4202,
              "end": 4247
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
                  "start": 4259,
                  "end": 4266
                }
              },
              "loc": {
                "start": 4259,
                "end": 4266
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
                      "start": 4280,
                      "end": 4291
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4277,
                    "end": 4291
                  }
                }
              ],
              "loc": {
                "start": 4267,
                "end": 4297
              }
            },
            "loc": {
              "start": 4252,
              "end": 4297
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
                  "start": 4309,
                  "end": 4317
                }
              },
              "loc": {
                "start": 4309,
                "end": 4317
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
                      "start": 4331,
                      "end": 4343
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4328,
                    "end": 4343
                  }
                }
              ],
              "loc": {
                "start": 4318,
                "end": 4349
              }
            },
            "loc": {
              "start": 4302,
              "end": 4349
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
                  "start": 4361,
                  "end": 4365
                }
              },
              "loc": {
                "start": 4361,
                "end": 4365
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
                      "start": 4379,
                      "end": 4387
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4376,
                    "end": 4387
                  }
                }
              ],
              "loc": {
                "start": 4366,
                "end": 4393
              }
            },
            "loc": {
              "start": 4354,
              "end": 4393
            }
          }
        ],
        "loc": {
          "start": 4066,
          "end": 4395
        }
      },
      "loc": {
        "start": 4056,
        "end": 4395
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4396,
          "end": 4400
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
                "start": 4410,
                "end": 4418
              }
            },
            "directives": [],
            "loc": {
              "start": 4407,
              "end": 4418
            }
          }
        ],
        "loc": {
          "start": 4401,
          "end": 4420
        }
      },
      "loc": {
        "start": 4396,
        "end": 4420
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4421,
          "end": 4424
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
                "start": 4431,
                "end": 4439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4431,
              "end": 4439
            }
          }
        ],
        "loc": {
          "start": 4425,
          "end": 4441
        }
      },
      "loc": {
        "start": 4421,
        "end": 4441
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 4442,
          "end": 4454
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
                "start": 4461,
                "end": 4463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4461,
              "end": 4463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 4468,
                "end": 4476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4468,
              "end": 4476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 4481,
                "end": 4492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4481,
              "end": 4492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 4497,
                "end": 4501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4497,
              "end": 4501
            }
          }
        ],
        "loc": {
          "start": 4455,
          "end": 4503
        }
      },
      "loc": {
        "start": 4442,
        "end": 4503
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4541,
          "end": 4543
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4541,
        "end": 4543
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4544,
          "end": 4554
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4544,
        "end": 4554
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4555,
          "end": 4565
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4555,
        "end": 4565
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 4566,
          "end": 4576
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4566,
        "end": 4576
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4577,
          "end": 4586
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4577,
        "end": 4586
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 4587,
          "end": 4598
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4587,
        "end": 4598
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 4599,
          "end": 4605
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
                "start": 4615,
                "end": 4625
              }
            },
            "directives": [],
            "loc": {
              "start": 4612,
              "end": 4625
            }
          }
        ],
        "loc": {
          "start": 4606,
          "end": 4627
        }
      },
      "loc": {
        "start": 4599,
        "end": 4627
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 4628,
          "end": 4633
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
                  "start": 4647,
                  "end": 4651
                }
              },
              "loc": {
                "start": 4647,
                "end": 4651
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
                      "start": 4665,
                      "end": 4673
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4662,
                    "end": 4673
                  }
                }
              ],
              "loc": {
                "start": 4652,
                "end": 4679
              }
            },
            "loc": {
              "start": 4640,
              "end": 4679
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
                  "start": 4691,
                  "end": 4695
                }
              },
              "loc": {
                "start": 4691,
                "end": 4695
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
                      "start": 4709,
                      "end": 4717
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4706,
                    "end": 4717
                  }
                }
              ],
              "loc": {
                "start": 4696,
                "end": 4723
              }
            },
            "loc": {
              "start": 4684,
              "end": 4723
            }
          }
        ],
        "loc": {
          "start": 4634,
          "end": 4725
        }
      },
      "loc": {
        "start": 4628,
        "end": 4725
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 4726,
          "end": 4737
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4726,
        "end": 4737
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 4738,
          "end": 4752
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4738,
        "end": 4752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4753,
          "end": 4758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4753,
        "end": 4758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4759,
          "end": 4768
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4759,
        "end": 4768
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4769,
          "end": 4773
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
                "start": 4783,
                "end": 4791
              }
            },
            "directives": [],
            "loc": {
              "start": 4780,
              "end": 4791
            }
          }
        ],
        "loc": {
          "start": 4774,
          "end": 4793
        }
      },
      "loc": {
        "start": 4769,
        "end": 4793
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 4794,
          "end": 4808
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4794,
        "end": 4808
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 4809,
          "end": 4814
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4809,
        "end": 4814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4815,
          "end": 4818
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
                "start": 4825,
                "end": 4835
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4825,
              "end": 4835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 4840,
                "end": 4849
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4840,
              "end": 4849
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 4854,
                "end": 4865
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4854,
              "end": 4865
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 4870,
                "end": 4879
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4870,
              "end": 4879
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 4884,
                "end": 4891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4884,
              "end": 4891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 4896,
                "end": 4904
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4896,
              "end": 4904
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 4909,
                "end": 4921
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4909,
              "end": 4921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 4926,
                "end": 4934
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4926,
              "end": 4934
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 4939,
                "end": 4947
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4939,
              "end": 4947
            }
          }
        ],
        "loc": {
          "start": 4819,
          "end": 4949
        }
      },
      "loc": {
        "start": 4815,
        "end": 4949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4950,
          "end": 4958
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
                "start": 4965,
                "end": 4967
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4965,
              "end": 4967
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4972,
                "end": 4982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4972,
              "end": 4982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4987,
                "end": 4997
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4987,
              "end": 4997
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 5002,
                "end": 5013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5002,
              "end": 5013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 5018,
                "end": 5031
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5018,
              "end": 5031
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
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
              "value": "isDeleted",
              "loc": {
                "start": 5051,
                "end": 5060
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5051,
              "end": 5060
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5065,
                "end": 5073
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5065,
              "end": 5073
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5078,
                "end": 5087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5078,
              "end": 5087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineType",
              "loc": {
                "start": 5092,
                "end": 5103
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5092,
              "end": 5103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 5108,
                "end": 5118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5108,
              "end": 5118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 5123,
                "end": 5135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5123,
              "end": 5135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 5140,
                "end": 5154
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5140,
              "end": 5154
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5159,
                "end": 5171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5159,
              "end": 5171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5176,
                "end": 5188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5176,
              "end": 5188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5193,
                "end": 5206
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5193,
              "end": 5206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5211,
                "end": 5233
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5211,
              "end": 5233
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5238,
                "end": 5248
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5238,
              "end": 5248
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 5253,
                "end": 5264
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5253,
              "end": 5264
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 5269,
                "end": 5279
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5269,
              "end": 5279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 5284,
                "end": 5298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5284,
              "end": 5298
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 5303,
                "end": 5315
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5303,
              "end": 5315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5320,
                "end": 5332
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5320,
              "end": 5332
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 5337,
                "end": 5349
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
                      "start": 5360,
                      "end": 5362
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5360,
                    "end": 5362
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5371,
                      "end": 5379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5371,
                    "end": 5379
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5388,
                      "end": 5399
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5388,
                    "end": 5399
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 5408,
                      "end": 5420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5408,
                    "end": 5420
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5429,
                      "end": 5433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5429,
                    "end": 5433
                  }
                }
              ],
              "loc": {
                "start": 5350,
                "end": 5439
              }
            },
            "loc": {
              "start": 5337,
              "end": 5439
            }
          }
        ],
        "loc": {
          "start": 4959,
          "end": 5441
        }
      },
      "loc": {
        "start": 4950,
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
        "value": "id",
        "loc": {
          "start": 5541,
          "end": 5543
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5541,
        "end": 5543
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5544,
          "end": 5554
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5544,
        "end": 5554
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5555,
          "end": 5565
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5555,
        "end": 5565
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5566,
          "end": 5575
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5566,
        "end": 5575
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5576,
          "end": 5587
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5576,
        "end": 5587
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5588,
          "end": 5594
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
                "start": 5604,
                "end": 5614
              }
            },
            "directives": [],
            "loc": {
              "start": 5601,
              "end": 5614
            }
          }
        ],
        "loc": {
          "start": 5595,
          "end": 5616
        }
      },
      "loc": {
        "start": 5588,
        "end": 5616
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5617,
          "end": 5622
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
                  "start": 5636,
                  "end": 5640
                }
              },
              "loc": {
                "start": 5636,
                "end": 5640
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
                      "start": 5654,
                      "end": 5662
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5651,
                    "end": 5662
                  }
                }
              ],
              "loc": {
                "start": 5641,
                "end": 5668
              }
            },
            "loc": {
              "start": 5629,
              "end": 5668
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
                  "start": 5680,
                  "end": 5684
                }
              },
              "loc": {
                "start": 5680,
                "end": 5684
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
                      "start": 5698,
                      "end": 5706
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5695,
                    "end": 5706
                  }
                }
              ],
              "loc": {
                "start": 5685,
                "end": 5712
              }
            },
            "loc": {
              "start": 5673,
              "end": 5712
            }
          }
        ],
        "loc": {
          "start": 5623,
          "end": 5714
        }
      },
      "loc": {
        "start": 5617,
        "end": 5714
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5715,
          "end": 5726
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5715,
        "end": 5726
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5727,
          "end": 5741
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5727,
        "end": 5741
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5742,
          "end": 5747
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5742,
        "end": 5747
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5748,
          "end": 5757
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5748,
        "end": 5757
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5758,
          "end": 5762
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
                "start": 5772,
                "end": 5780
              }
            },
            "directives": [],
            "loc": {
              "start": 5769,
              "end": 5780
            }
          }
        ],
        "loc": {
          "start": 5763,
          "end": 5782
        }
      },
      "loc": {
        "start": 5758,
        "end": 5782
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5783,
          "end": 5797
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5783,
        "end": 5797
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5798,
          "end": 5803
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5798,
        "end": 5803
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5804,
          "end": 5807
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
                "start": 5814,
                "end": 5823
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5814,
              "end": 5823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5828,
                "end": 5839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5828,
              "end": 5839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 5844,
                "end": 5855
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5844,
              "end": 5855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5860,
                "end": 5869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5860,
              "end": 5869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5874,
                "end": 5881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5874,
              "end": 5881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5886,
                "end": 5894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5886,
              "end": 5894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5899,
                "end": 5911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5899,
              "end": 5911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5916,
                "end": 5924
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5916,
              "end": 5924
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5929,
                "end": 5937
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5929,
              "end": 5937
            }
          }
        ],
        "loc": {
          "start": 5808,
          "end": 5939
        }
      },
      "loc": {
        "start": 5804,
        "end": 5939
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5940,
          "end": 5948
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
                "start": 5955,
                "end": 5957
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5955,
              "end": 5957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5962,
                "end": 5972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5962,
              "end": 5972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5977,
                "end": 5987
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5977,
              "end": 5987
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5992,
                "end": 6002
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5992,
              "end": 6002
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 6007,
                "end": 6013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6007,
              "end": 6013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6018,
                "end": 6026
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6018,
              "end": 6026
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6031,
                "end": 6040
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6031,
              "end": 6040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6045,
                "end": 6052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6045,
              "end": 6052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 6057,
                "end": 6069
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6057,
              "end": 6069
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 6074,
                "end": 6079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6074,
              "end": 6079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 6084,
                "end": 6087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6084,
              "end": 6087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6092,
                "end": 6104
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6092,
              "end": 6104
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6109,
                "end": 6121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6109,
              "end": 6121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6126,
                "end": 6139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6126,
              "end": 6139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6144,
                "end": 6166
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6144,
              "end": 6166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6171,
                "end": 6181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6171,
              "end": 6181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6186,
                "end": 6198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6186,
              "end": 6198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6203,
                "end": 6206
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
                      "start": 6217,
                      "end": 6227
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6217,
                    "end": 6227
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6236,
                      "end": 6243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6236,
                    "end": 6243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
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
                    "value": "canReport",
                    "loc": {
                      "start": 6270,
                      "end": 6279
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6270,
                    "end": 6279
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6288,
                      "end": 6297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6288,
                    "end": 6297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6306,
                      "end": 6312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6306,
                    "end": 6312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6321,
                      "end": 6328
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6321,
                    "end": 6328
                  }
                }
              ],
              "loc": {
                "start": 6207,
                "end": 6334
              }
            },
            "loc": {
              "start": 6203,
              "end": 6334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6339,
                "end": 6351
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
                      "start": 6362,
                      "end": 6364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6362,
                    "end": 6364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6373,
                      "end": 6381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6373,
                    "end": 6381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6390,
                      "end": 6401
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6390,
                    "end": 6401
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6410,
                      "end": 6422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6410,
                    "end": 6422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6431,
                      "end": 6435
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6431,
                    "end": 6435
                  }
                }
              ],
              "loc": {
                "start": 6352,
                "end": 6441
              }
            },
            "loc": {
              "start": 6339,
              "end": 6441
            }
          }
        ],
        "loc": {
          "start": 5949,
          "end": 6443
        }
      },
      "loc": {
        "start": 5940,
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
        "value": "id",
        "loc": {
          "start": 7458,
          "end": 7460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7458,
        "end": 7460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7461,
          "end": 7471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7461,
        "end": 7471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7472,
          "end": 7482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7472,
        "end": 7482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7483,
          "end": 7494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7483,
        "end": 7494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7495,
          "end": 7501
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7495,
        "end": 7501
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7502,
          "end": 7507
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7502,
        "end": 7507
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7508,
          "end": 7528
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7508,
        "end": 7528
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7529,
          "end": 7533
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7529,
        "end": 7533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7534,
          "end": 7546
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7534,
        "end": 7546
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7547,
          "end": 7556
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7547,
        "end": 7556
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7557,
          "end": 7577
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7557,
        "end": 7577
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7578,
          "end": 7581
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
                "start": 7588,
                "end": 7597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7588,
              "end": 7597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7602,
                "end": 7611
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7602,
              "end": 7611
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7616,
                "end": 7625
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7616,
              "end": 7625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7630,
                "end": 7642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7630,
              "end": 7642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7647,
                "end": 7655
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7647,
              "end": 7655
            }
          }
        ],
        "loc": {
          "start": 7582,
          "end": 7657
        }
      },
      "loc": {
        "start": 7578,
        "end": 7657
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7658,
          "end": 7670
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
                "start": 7677,
                "end": 7679
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7677,
              "end": 7679
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7684,
                "end": 7692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7684,
              "end": 7692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7697,
                "end": 7700
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7697,
              "end": 7700
            }
          }
        ],
        "loc": {
          "start": 7671,
          "end": 7702
        }
      },
      "loc": {
        "start": 7658,
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
              "value": "id",
              "loc": {
                "start": 28,
                "end": 30
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 28,
              "end": 30
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 31,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 31,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 42,
                "end": 52
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 52
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 53,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 53,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 63,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 74
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 75,
                "end": 81
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
                      "start": 91,
                      "end": 101
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 88,
                    "end": 101
                  }
                }
              ],
              "loc": {
                "start": 82,
                "end": 103
              }
            },
            "loc": {
              "start": 75,
              "end": 103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 104,
                "end": 109
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
                        "start": 123,
                        "end": 127
                      }
                    },
                    "loc": {
                      "start": 123,
                      "end": 127
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
                            "start": 141,
                            "end": 149
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 138,
                          "end": 149
                        }
                      }
                    ],
                    "loc": {
                      "start": 128,
                      "end": 155
                    }
                  },
                  "loc": {
                    "start": 116,
                    "end": 155
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
                        "start": 167,
                        "end": 171
                      }
                    },
                    "loc": {
                      "start": 167,
                      "end": 171
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
                            "start": 185,
                            "end": 193
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 182,
                          "end": 193
                        }
                      }
                    ],
                    "loc": {
                      "start": 172,
                      "end": 199
                    }
                  },
                  "loc": {
                    "start": 160,
                    "end": 199
                  }
                }
              ],
              "loc": {
                "start": 110,
                "end": 201
              }
            },
            "loc": {
              "start": 104,
              "end": 201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 202,
                "end": 213
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 202,
              "end": 213
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 214,
                "end": 228
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 214,
              "end": 228
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 229,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 229,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 235,
                "end": 244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 235,
              "end": 244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 245,
                "end": 249
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
                      "start": 259,
                      "end": 267
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 256,
                    "end": 267
                  }
                }
              ],
              "loc": {
                "start": 250,
                "end": 269
              }
            },
            "loc": {
              "start": 245,
              "end": 269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 270,
                "end": 284
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 270,
              "end": 284
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 285,
                "end": 290
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 285,
              "end": 290
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 291,
                "end": 294
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
                      "start": 301,
                      "end": 310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 301,
                    "end": 310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 315,
                      "end": 326
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 315,
                    "end": 326
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 331,
                      "end": 342
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 331,
                    "end": 342
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 347,
                      "end": 356
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 347,
                    "end": 356
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 361,
                      "end": 368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 361,
                    "end": 368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 373,
                      "end": 381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 386,
                      "end": 398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 386,
                    "end": 398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 403,
                      "end": 411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 403,
                    "end": 411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 416,
                      "end": 424
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 416,
                    "end": 424
                  }
                }
              ],
              "loc": {
                "start": 295,
                "end": 426
              }
            },
            "loc": {
              "start": 291,
              "end": 426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 427,
                "end": 435
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
                      "start": 442,
                      "end": 444
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 444
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
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
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 464,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 464,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 479,
                      "end": 487
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 479,
                    "end": 487
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 492,
                      "end": 505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 492,
                    "end": 505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 510,
                      "end": 527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 510,
                    "end": 527
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 532,
                      "end": 542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 532,
                    "end": 542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 547,
                      "end": 555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 547,
                    "end": 555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 560,
                      "end": 569
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 560,
                    "end": 569
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 574,
                      "end": 586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 574,
                    "end": 586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 591,
                      "end": 603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 591,
                    "end": 603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 608,
                      "end": 620
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 608,
                    "end": 620
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 625,
                      "end": 628
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
                            "start": 639,
                            "end": 649
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 639,
                          "end": 649
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 658,
                            "end": 665
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 658,
                          "end": 665
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 674,
                            "end": 683
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 674,
                          "end": 683
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 692,
                            "end": 701
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 692,
                          "end": 701
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 710,
                            "end": 719
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 710,
                          "end": 719
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 728,
                            "end": 734
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 728,
                          "end": 734
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 743,
                            "end": 750
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 743,
                          "end": 750
                        }
                      }
                    ],
                    "loc": {
                      "start": 629,
                      "end": 756
                    }
                  },
                  "loc": {
                    "start": 625,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 761,
                      "end": 773
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
                            "start": 784,
                            "end": 786
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 784,
                          "end": 786
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 795,
                            "end": 803
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 795,
                          "end": 803
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 812,
                            "end": 819
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 812,
                          "end": 819
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 828,
                            "end": 832
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 828,
                          "end": 832
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 841,
                            "end": 848
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 841,
                          "end": 848
                        }
                      }
                    ],
                    "loc": {
                      "start": 774,
                      "end": 854
                    }
                  },
                  "loc": {
                    "start": 761,
                    "end": 854
                  }
                }
              ],
              "loc": {
                "start": 436,
                "end": 856
              }
            },
            "loc": {
              "start": 427,
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
              "value": "id",
              "loc": {
                "start": 929,
                "end": 931
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 929,
              "end": 931
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 932,
                "end": 942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 932,
              "end": 942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 943,
                "end": 953
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 943,
              "end": 953
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 954,
                "end": 963
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 954,
              "end": 963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 964,
                "end": 975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 964,
              "end": 975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 976,
                "end": 982
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
                      "start": 992,
                      "end": 1002
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 989,
                    "end": 1002
                  }
                }
              ],
              "loc": {
                "start": 983,
                "end": 1004
              }
            },
            "loc": {
              "start": 976,
              "end": 1004
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1005,
                "end": 1010
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
                        "start": 1024,
                        "end": 1028
                      }
                    },
                    "loc": {
                      "start": 1024,
                      "end": 1028
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
                            "start": 1042,
                            "end": 1050
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1039,
                          "end": 1050
                        }
                      }
                    ],
                    "loc": {
                      "start": 1029,
                      "end": 1056
                    }
                  },
                  "loc": {
                    "start": 1017,
                    "end": 1056
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
                        "start": 1068,
                        "end": 1072
                      }
                    },
                    "loc": {
                      "start": 1068,
                      "end": 1072
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
                            "start": 1086,
                            "end": 1094
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1083,
                          "end": 1094
                        }
                      }
                    ],
                    "loc": {
                      "start": 1073,
                      "end": 1100
                    }
                  },
                  "loc": {
                    "start": 1061,
                    "end": 1100
                  }
                }
              ],
              "loc": {
                "start": 1011,
                "end": 1102
              }
            },
            "loc": {
              "start": 1005,
              "end": 1102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1103,
                "end": 1114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1103,
              "end": 1114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1115,
                "end": 1129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1115,
              "end": 1129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1130,
                "end": 1135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1130,
              "end": 1135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1136,
                "end": 1145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1136,
              "end": 1145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1146,
                "end": 1150
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
                      "start": 1160,
                      "end": 1168
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1157,
                    "end": 1168
                  }
                }
              ],
              "loc": {
                "start": 1151,
                "end": 1170
              }
            },
            "loc": {
              "start": 1146,
              "end": 1170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1171,
                "end": 1185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1171,
              "end": 1185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1186,
                "end": 1191
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1191
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1192,
                "end": 1195
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
                      "start": 1202,
                      "end": 1211
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1202,
                    "end": 1211
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1216,
                      "end": 1227
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1216,
                    "end": 1227
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1232,
                      "end": 1243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1232,
                    "end": 1243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1248,
                      "end": 1257
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1257
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1262,
                      "end": 1269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1262,
                    "end": 1269
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1274,
                      "end": 1282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1274,
                    "end": 1282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1287,
                      "end": 1299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1287,
                    "end": 1299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1304,
                      "end": 1312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1304,
                    "end": 1312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1317,
                      "end": 1325
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1317,
                    "end": 1325
                  }
                }
              ],
              "loc": {
                "start": 1196,
                "end": 1327
              }
            },
            "loc": {
              "start": 1192,
              "end": 1327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1328,
                "end": 1336
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
                      "start": 1343,
                      "end": 1345
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1343,
                    "end": 1345
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1350,
                      "end": 1360
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1350,
                    "end": 1360
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1365,
                      "end": 1375
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1365,
                    "end": 1375
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1380,
                      "end": 1390
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1380,
                    "end": 1390
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1395,
                      "end": 1404
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1395,
                    "end": 1404
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1409,
                      "end": 1417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1409,
                    "end": 1417
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1422,
                      "end": 1431
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1422,
                    "end": 1431
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeLanguage",
                    "loc": {
                      "start": 1436,
                      "end": 1448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1436,
                    "end": 1448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeType",
                    "loc": {
                      "start": 1453,
                      "end": 1461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1453,
                    "end": 1461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 1466,
                      "end": 1473
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1466,
                    "end": 1473
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1478,
                      "end": 1490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1478,
                    "end": 1490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1495,
                      "end": 1507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1495,
                    "end": 1507
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "calledByRoutineVersionsCount",
                    "loc": {
                      "start": 1512,
                      "end": 1540
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1512,
                    "end": 1540
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1545,
                      "end": 1558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1545,
                    "end": 1558
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1563,
                      "end": 1585
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1563,
                    "end": 1585
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1590,
                      "end": 1600
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1590,
                    "end": 1600
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1605,
                      "end": 1617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1605,
                    "end": 1617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1622,
                      "end": 1625
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
                            "start": 1636,
                            "end": 1646
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1636,
                          "end": 1646
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1655,
                            "end": 1662
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1655,
                          "end": 1662
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
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
                          "value": "canReport",
                          "loc": {
                            "start": 1689,
                            "end": 1698
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1689,
                          "end": 1698
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1707,
                            "end": 1716
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1707,
                          "end": 1716
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1725,
                            "end": 1731
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1725,
                          "end": 1731
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1740,
                            "end": 1747
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1740,
                          "end": 1747
                        }
                      }
                    ],
                    "loc": {
                      "start": 1626,
                      "end": 1753
                    }
                  },
                  "loc": {
                    "start": 1622,
                    "end": 1753
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1758,
                      "end": 1770
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
                            "start": 1781,
                            "end": 1783
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1781,
                          "end": 1783
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1792,
                            "end": 1800
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1792,
                          "end": 1800
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1809,
                            "end": 1820
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1809,
                          "end": 1820
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 1829,
                            "end": 1841
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1829,
                          "end": 1841
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1850,
                            "end": 1854
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1850,
                          "end": 1854
                        }
                      }
                    ],
                    "loc": {
                      "start": 1771,
                      "end": 1860
                    }
                  },
                  "loc": {
                    "start": 1758,
                    "end": 1860
                  }
                }
              ],
              "loc": {
                "start": 1337,
                "end": 1862
              }
            },
            "loc": {
              "start": 1328,
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
              "value": "id",
              "loc": {
                "start": 2141,
                "end": 2143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2141,
              "end": 2143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2144,
                "end": 2154
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2144,
              "end": 2154
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2155,
                "end": 2165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2155,
              "end": 2165
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2166,
                "end": 2175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2166,
              "end": 2175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2176,
                "end": 2187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2176,
              "end": 2187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2188,
                "end": 2194
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
                      "start": 2204,
                      "end": 2214
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2201,
                    "end": 2214
                  }
                }
              ],
              "loc": {
                "start": 2195,
                "end": 2216
              }
            },
            "loc": {
              "start": 2188,
              "end": 2216
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2217,
                "end": 2222
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
                        "start": 2236,
                        "end": 2240
                      }
                    },
                    "loc": {
                      "start": 2236,
                      "end": 2240
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
                            "start": 2254,
                            "end": 2262
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2251,
                          "end": 2262
                        }
                      }
                    ],
                    "loc": {
                      "start": 2241,
                      "end": 2268
                    }
                  },
                  "loc": {
                    "start": 2229,
                    "end": 2268
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
                        "start": 2280,
                        "end": 2284
                      }
                    },
                    "loc": {
                      "start": 2280,
                      "end": 2284
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
                            "start": 2298,
                            "end": 2306
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2295,
                          "end": 2306
                        }
                      }
                    ],
                    "loc": {
                      "start": 2285,
                      "end": 2312
                    }
                  },
                  "loc": {
                    "start": 2273,
                    "end": 2312
                  }
                }
              ],
              "loc": {
                "start": 2223,
                "end": 2314
              }
            },
            "loc": {
              "start": 2217,
              "end": 2314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2315,
                "end": 2326
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2315,
              "end": 2326
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2327,
                "end": 2341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2327,
              "end": 2341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2342,
                "end": 2347
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2342,
              "end": 2347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2348,
                "end": 2357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2348,
              "end": 2357
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2358,
                "end": 2362
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
                      "start": 2372,
                      "end": 2380
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2369,
                    "end": 2380
                  }
                }
              ],
              "loc": {
                "start": 2363,
                "end": 2382
              }
            },
            "loc": {
              "start": 2358,
              "end": 2382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2383,
                "end": 2397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2383,
              "end": 2397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2398,
                "end": 2403
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2398,
              "end": 2403
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2404,
                "end": 2407
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
                      "start": 2414,
                      "end": 2423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2414,
                    "end": 2423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2428,
                      "end": 2439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2428,
                    "end": 2439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2444,
                      "end": 2455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2444,
                    "end": 2455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2460,
                      "end": 2469
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2460,
                    "end": 2469
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2474,
                      "end": 2481
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2474,
                    "end": 2481
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2486,
                      "end": 2494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2486,
                    "end": 2494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2499,
                      "end": 2511
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2499,
                    "end": 2511
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2516,
                      "end": 2524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2516,
                    "end": 2524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2529,
                      "end": 2537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2529,
                    "end": 2537
                  }
                }
              ],
              "loc": {
                "start": 2408,
                "end": 2539
              }
            },
            "loc": {
              "start": 2404,
              "end": 2539
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 2540,
                "end": 2548
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
                      "start": 2555,
                      "end": 2557
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2555,
                    "end": 2557
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2562,
                      "end": 2572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2562,
                    "end": 2572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2577,
                      "end": 2587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2577,
                    "end": 2587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 2592,
                      "end": 2600
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2592,
                    "end": 2600
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 2605,
                      "end": 2614
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2605,
                    "end": 2614
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2619,
                      "end": 2631
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2619,
                    "end": 2631
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 2636,
                      "end": 2648
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2636,
                    "end": 2648
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 2653,
                      "end": 2665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2653,
                    "end": 2665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2670,
                      "end": 2673
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
                            "start": 2684,
                            "end": 2694
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2684,
                          "end": 2694
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 2703,
                            "end": 2710
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2703,
                          "end": 2710
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2719,
                            "end": 2728
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2719,
                          "end": 2728
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2737,
                            "end": 2746
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2737,
                          "end": 2746
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2755,
                            "end": 2764
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2755,
                          "end": 2764
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 2773,
                            "end": 2779
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2773,
                          "end": 2779
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2788,
                            "end": 2795
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2788,
                          "end": 2795
                        }
                      }
                    ],
                    "loc": {
                      "start": 2674,
                      "end": 2801
                    }
                  },
                  "loc": {
                    "start": 2670,
                    "end": 2801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2806,
                      "end": 2818
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
                            "start": 2829,
                            "end": 2831
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2829,
                          "end": 2831
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2840,
                            "end": 2848
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2840,
                          "end": 2848
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2857,
                            "end": 2868
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2857,
                          "end": 2868
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2877,
                            "end": 2881
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2877,
                          "end": 2881
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 2890,
                            "end": 2895
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
                                  "start": 2910,
                                  "end": 2912
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2910,
                                "end": 2912
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 2925,
                                  "end": 2934
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2925,
                                "end": 2934
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 2947,
                                  "end": 2951
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2947,
                                "end": 2951
                              }
                            }
                          ],
                          "loc": {
                            "start": 2896,
                            "end": 2961
                          }
                        },
                        "loc": {
                          "start": 2890,
                          "end": 2961
                        }
                      }
                    ],
                    "loc": {
                      "start": 2819,
                      "end": 2967
                    }
                  },
                  "loc": {
                    "start": 2806,
                    "end": 2967
                  }
                }
              ],
              "loc": {
                "start": 2549,
                "end": 2969
              }
            },
            "loc": {
              "start": 2540,
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
              "value": "id",
              "loc": {
                "start": 3050,
                "end": 3052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3050,
              "end": 3052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3053,
                "end": 3063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3053,
              "end": 3063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3064,
                "end": 3074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3064,
              "end": 3074
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3075,
                "end": 3084
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3075,
              "end": 3084
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3085,
                "end": 3096
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3085,
              "end": 3096
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3097,
                "end": 3103
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
                      "start": 3113,
                      "end": 3123
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3110,
                    "end": 3123
                  }
                }
              ],
              "loc": {
                "start": 3104,
                "end": 3125
              }
            },
            "loc": {
              "start": 3097,
              "end": 3125
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3126,
                "end": 3131
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
                        "start": 3145,
                        "end": 3149
                      }
                    },
                    "loc": {
                      "start": 3145,
                      "end": 3149
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
                            "start": 3163,
                            "end": 3171
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3160,
                          "end": 3171
                        }
                      }
                    ],
                    "loc": {
                      "start": 3150,
                      "end": 3177
                    }
                  },
                  "loc": {
                    "start": 3138,
                    "end": 3177
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
                        "start": 3189,
                        "end": 3193
                      }
                    },
                    "loc": {
                      "start": 3189,
                      "end": 3193
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
                            "start": 3207,
                            "end": 3215
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3204,
                          "end": 3215
                        }
                      }
                    ],
                    "loc": {
                      "start": 3194,
                      "end": 3221
                    }
                  },
                  "loc": {
                    "start": 3182,
                    "end": 3221
                  }
                }
              ],
              "loc": {
                "start": 3132,
                "end": 3223
              }
            },
            "loc": {
              "start": 3126,
              "end": 3223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3224,
                "end": 3235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3224,
              "end": 3235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3236,
                "end": 3250
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3236,
              "end": 3250
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3251,
                "end": 3256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3251,
              "end": 3256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3257,
                "end": 3266
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3257,
              "end": 3266
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3267,
                "end": 3271
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
                      "start": 3281,
                      "end": 3289
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3278,
                    "end": 3289
                  }
                }
              ],
              "loc": {
                "start": 3272,
                "end": 3291
              }
            },
            "loc": {
              "start": 3267,
              "end": 3291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3292,
                "end": 3306
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3292,
              "end": 3306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3307,
                "end": 3312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3307,
              "end": 3312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3313,
                "end": 3316
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
                      "start": 3323,
                      "end": 3332
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3323,
                    "end": 3332
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3337,
                      "end": 3348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3337,
                    "end": 3348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3353,
                      "end": 3364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3353,
                    "end": 3364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3369,
                      "end": 3378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3369,
                    "end": 3378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3383,
                      "end": 3390
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3383,
                    "end": 3390
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3395,
                      "end": 3403
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3395,
                    "end": 3403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3408,
                      "end": 3420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3408,
                    "end": 3420
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3425,
                      "end": 3433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3425,
                    "end": 3433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3438,
                      "end": 3446
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3438,
                    "end": 3446
                  }
                }
              ],
              "loc": {
                "start": 3317,
                "end": 3448
              }
            },
            "loc": {
              "start": 3313,
              "end": 3448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 3449,
                "end": 3457
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
                      "start": 3464,
                      "end": 3466
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3464,
                    "end": 3466
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3471,
                      "end": 3481
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3471,
                    "end": 3481
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3486,
                      "end": 3496
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3486,
                    "end": 3496
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3501,
                      "end": 3517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3501,
                    "end": 3517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3522,
                      "end": 3530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3522,
                    "end": 3530
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3535,
                      "end": 3544
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3535,
                    "end": 3544
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3549,
                      "end": 3561
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3549,
                    "end": 3561
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3566,
                      "end": 3582
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3566,
                    "end": 3582
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3587,
                      "end": 3597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3587,
                    "end": 3597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3602,
                      "end": 3614
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3602,
                    "end": 3614
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3619,
                      "end": 3631
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3619,
                    "end": 3631
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3636,
                      "end": 3648
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
                            "start": 3659,
                            "end": 3661
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3659,
                          "end": 3661
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3670,
                            "end": 3678
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3670,
                          "end": 3678
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3687,
                            "end": 3698
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3687,
                          "end": 3698
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3707,
                            "end": 3711
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3707,
                          "end": 3711
                        }
                      }
                    ],
                    "loc": {
                      "start": 3649,
                      "end": 3717
                    }
                  },
                  "loc": {
                    "start": 3636,
                    "end": 3717
                  }
                }
              ],
              "loc": {
                "start": 3458,
                "end": 3719
              }
            },
            "loc": {
              "start": 3449,
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
              "value": "id",
              "loc": {
                "start": 3808,
                "end": 3810
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3808,
              "end": 3810
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3811,
                "end": 3821
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3811,
              "end": 3821
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3822,
                "end": 3832
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3822,
              "end": 3832
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3833,
                "end": 3842
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
                      "start": 3849,
                      "end": 3851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3849,
                    "end": 3851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3856,
                      "end": 3866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3856,
                    "end": 3866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3871,
                      "end": 3881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3871,
                    "end": 3881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3886,
                      "end": 3897
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3886,
                    "end": 3897
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3902,
                      "end": 3908
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3902,
                    "end": 3908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3913,
                      "end": 3918
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3913,
                    "end": 3918
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBotDepictingPerson",
                    "loc": {
                      "start": 3923,
                      "end": 3943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3923,
                    "end": 3943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3948,
                      "end": 3952
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3948,
                    "end": 3952
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3957,
                      "end": 3969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3957,
                    "end": 3969
                  }
                }
              ],
              "loc": {
                "start": 3843,
                "end": 3971
              }
            },
            "loc": {
              "start": 3833,
              "end": 3971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 3972,
                "end": 3989
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3972,
              "end": 3989
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3990,
                "end": 3999
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3990,
              "end": 3999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4000,
                "end": 4005
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4000,
              "end": 4005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4006,
                "end": 4015
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4006,
              "end": 4015
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4016,
                "end": 4028
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4016,
              "end": 4028
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4029,
                "end": 4042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4029,
              "end": 4042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4043,
                "end": 4055
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4043,
              "end": 4055
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4056,
                "end": 4065
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
                        "start": 4079,
                        "end": 4082
                      }
                    },
                    "loc": {
                      "start": 4079,
                      "end": 4082
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
                            "start": 4096,
                            "end": 4103
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4093,
                          "end": 4103
                        }
                      }
                    ],
                    "loc": {
                      "start": 4083,
                      "end": 4109
                    }
                  },
                  "loc": {
                    "start": 4072,
                    "end": 4109
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
                        "start": 4121,
                        "end": 4125
                      }
                    },
                    "loc": {
                      "start": 4121,
                      "end": 4125
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
                            "start": 4139,
                            "end": 4147
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4136,
                          "end": 4147
                        }
                      }
                    ],
                    "loc": {
                      "start": 4126,
                      "end": 4153
                    }
                  },
                  "loc": {
                    "start": 4114,
                    "end": 4153
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
                        "start": 4165,
                        "end": 4169
                      }
                    },
                    "loc": {
                      "start": 4165,
                      "end": 4169
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
                            "start": 4183,
                            "end": 4191
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4180,
                          "end": 4191
                        }
                      }
                    ],
                    "loc": {
                      "start": 4170,
                      "end": 4197
                    }
                  },
                  "loc": {
                    "start": 4158,
                    "end": 4197
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
                        "start": 4209,
                        "end": 4216
                      }
                    },
                    "loc": {
                      "start": 4209,
                      "end": 4216
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
                            "start": 4230,
                            "end": 4241
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4227,
                          "end": 4241
                        }
                      }
                    ],
                    "loc": {
                      "start": 4217,
                      "end": 4247
                    }
                  },
                  "loc": {
                    "start": 4202,
                    "end": 4247
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
                        "start": 4259,
                        "end": 4266
                      }
                    },
                    "loc": {
                      "start": 4259,
                      "end": 4266
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
                            "start": 4280,
                            "end": 4291
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4277,
                          "end": 4291
                        }
                      }
                    ],
                    "loc": {
                      "start": 4267,
                      "end": 4297
                    }
                  },
                  "loc": {
                    "start": 4252,
                    "end": 4297
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
                        "start": 4309,
                        "end": 4317
                      }
                    },
                    "loc": {
                      "start": 4309,
                      "end": 4317
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
                            "start": 4331,
                            "end": 4343
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4328,
                          "end": 4343
                        }
                      }
                    ],
                    "loc": {
                      "start": 4318,
                      "end": 4349
                    }
                  },
                  "loc": {
                    "start": 4302,
                    "end": 4349
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
                        "start": 4361,
                        "end": 4365
                      }
                    },
                    "loc": {
                      "start": 4361,
                      "end": 4365
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
                            "start": 4379,
                            "end": 4387
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4376,
                          "end": 4387
                        }
                      }
                    ],
                    "loc": {
                      "start": 4366,
                      "end": 4393
                    }
                  },
                  "loc": {
                    "start": 4354,
                    "end": 4393
                  }
                }
              ],
              "loc": {
                "start": 4066,
                "end": 4395
              }
            },
            "loc": {
              "start": 4056,
              "end": 4395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4396,
                "end": 4400
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
                      "start": 4410,
                      "end": 4418
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4407,
                    "end": 4418
                  }
                }
              ],
              "loc": {
                "start": 4401,
                "end": 4420
              }
            },
            "loc": {
              "start": 4396,
              "end": 4420
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4421,
                "end": 4424
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
                      "start": 4431,
                      "end": 4439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4431,
                    "end": 4439
                  }
                }
              ],
              "loc": {
                "start": 4425,
                "end": 4441
              }
            },
            "loc": {
              "start": 4421,
              "end": 4441
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 4442,
                "end": 4454
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
                      "start": 4461,
                      "end": 4463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4461,
                    "end": 4463
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4468,
                      "end": 4476
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4468,
                    "end": 4476
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4481,
                      "end": 4492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4481,
                    "end": 4492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4497,
                      "end": 4501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4497,
                    "end": 4501
                  }
                }
              ],
              "loc": {
                "start": 4455,
                "end": 4503
              }
            },
            "loc": {
              "start": 4442,
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
              "value": "id",
              "loc": {
                "start": 4541,
                "end": 4543
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4541,
              "end": 4543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4544,
                "end": 4554
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4544,
              "end": 4554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4555,
                "end": 4565
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4555,
              "end": 4565
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 4566,
                "end": 4576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4566,
              "end": 4576
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4577,
                "end": 4586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4577,
              "end": 4586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 4587,
                "end": 4598
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4587,
              "end": 4598
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 4599,
                "end": 4605
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
                      "start": 4615,
                      "end": 4625
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4612,
                    "end": 4625
                  }
                }
              ],
              "loc": {
                "start": 4606,
                "end": 4627
              }
            },
            "loc": {
              "start": 4599,
              "end": 4627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 4628,
                "end": 4633
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
                        "start": 4647,
                        "end": 4651
                      }
                    },
                    "loc": {
                      "start": 4647,
                      "end": 4651
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
                            "start": 4665,
                            "end": 4673
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4662,
                          "end": 4673
                        }
                      }
                    ],
                    "loc": {
                      "start": 4652,
                      "end": 4679
                    }
                  },
                  "loc": {
                    "start": 4640,
                    "end": 4679
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
                        "start": 4691,
                        "end": 4695
                      }
                    },
                    "loc": {
                      "start": 4691,
                      "end": 4695
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
                            "start": 4709,
                            "end": 4717
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4706,
                          "end": 4717
                        }
                      }
                    ],
                    "loc": {
                      "start": 4696,
                      "end": 4723
                    }
                  },
                  "loc": {
                    "start": 4684,
                    "end": 4723
                  }
                }
              ],
              "loc": {
                "start": 4634,
                "end": 4725
              }
            },
            "loc": {
              "start": 4628,
              "end": 4725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 4726,
                "end": 4737
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4726,
              "end": 4737
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 4738,
                "end": 4752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4738,
              "end": 4752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4753,
                "end": 4758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4753,
              "end": 4758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4759,
                "end": 4768
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4759,
              "end": 4768
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4769,
                "end": 4773
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
                      "start": 4783,
                      "end": 4791
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4780,
                    "end": 4791
                  }
                }
              ],
              "loc": {
                "start": 4774,
                "end": 4793
              }
            },
            "loc": {
              "start": 4769,
              "end": 4793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 4794,
                "end": 4808
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4794,
              "end": 4808
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 4809,
                "end": 4814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4809,
              "end": 4814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4815,
                "end": 4818
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
                      "start": 4825,
                      "end": 4835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4825,
                    "end": 4835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 4840,
                      "end": 4849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4840,
                    "end": 4849
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 4854,
                      "end": 4865
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4854,
                    "end": 4865
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 4870,
                      "end": 4879
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4870,
                    "end": 4879
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 4884,
                      "end": 4891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4884,
                    "end": 4891
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 4896,
                      "end": 4904
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4896,
                    "end": 4904
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 4909,
                      "end": 4921
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4909,
                    "end": 4921
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 4926,
                      "end": 4934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4926,
                    "end": 4934
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 4939,
                      "end": 4947
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4939,
                    "end": 4947
                  }
                }
              ],
              "loc": {
                "start": 4819,
                "end": 4949
              }
            },
            "loc": {
              "start": 4815,
              "end": 4949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4950,
                "end": 4958
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
                      "start": 4965,
                      "end": 4967
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4965,
                    "end": 4967
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4972,
                      "end": 4982
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4972,
                    "end": 4982
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4987,
                      "end": 4997
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4987,
                    "end": 4997
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 5002,
                      "end": 5013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5002,
                    "end": 5013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 5018,
                      "end": 5031
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5018,
                    "end": 5031
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
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
                    "value": "isDeleted",
                    "loc": {
                      "start": 5051,
                      "end": 5060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5051,
                    "end": 5060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5065,
                      "end": 5073
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5065,
                    "end": 5073
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5078,
                      "end": 5087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5078,
                    "end": 5087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 5092,
                      "end": 5103
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5092,
                    "end": 5103
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 5108,
                      "end": 5118
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5108,
                    "end": 5118
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 5123,
                      "end": 5135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5123,
                    "end": 5135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 5140,
                      "end": 5154
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5140,
                    "end": 5154
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5159,
                      "end": 5171
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5159,
                    "end": 5171
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5176,
                      "end": 5188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5176,
                    "end": 5188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5193,
                      "end": 5206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5193,
                    "end": 5206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5211,
                      "end": 5233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5211,
                    "end": 5233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5238,
                      "end": 5248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5238,
                    "end": 5248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 5253,
                      "end": 5264
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5253,
                    "end": 5264
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 5269,
                      "end": 5279
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5269,
                    "end": 5279
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 5284,
                      "end": 5298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5284,
                    "end": 5298
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 5303,
                      "end": 5315
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5303,
                    "end": 5315
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5320,
                      "end": 5332
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5320,
                    "end": 5332
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5337,
                      "end": 5349
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
                            "start": 5360,
                            "end": 5362
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5360,
                          "end": 5362
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5371,
                            "end": 5379
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5371,
                          "end": 5379
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5388,
                            "end": 5399
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5388,
                          "end": 5399
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 5408,
                            "end": 5420
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5408,
                          "end": 5420
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5429,
                            "end": 5433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5429,
                          "end": 5433
                        }
                      }
                    ],
                    "loc": {
                      "start": 5350,
                      "end": 5439
                    }
                  },
                  "loc": {
                    "start": 5337,
                    "end": 5439
                  }
                }
              ],
              "loc": {
                "start": 4959,
                "end": 5441
              }
            },
            "loc": {
              "start": 4950,
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
              "value": "id",
              "loc": {
                "start": 5541,
                "end": 5543
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5541,
              "end": 5543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5544,
                "end": 5554
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5544,
              "end": 5554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5555,
                "end": 5565
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5555,
              "end": 5565
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5566,
                "end": 5575
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5566,
              "end": 5575
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5576,
                "end": 5587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5576,
              "end": 5587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5588,
                "end": 5594
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
                      "start": 5604,
                      "end": 5614
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5601,
                    "end": 5614
                  }
                }
              ],
              "loc": {
                "start": 5595,
                "end": 5616
              }
            },
            "loc": {
              "start": 5588,
              "end": 5616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5617,
                "end": 5622
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
                        "start": 5636,
                        "end": 5640
                      }
                    },
                    "loc": {
                      "start": 5636,
                      "end": 5640
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
                            "start": 5654,
                            "end": 5662
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5651,
                          "end": 5662
                        }
                      }
                    ],
                    "loc": {
                      "start": 5641,
                      "end": 5668
                    }
                  },
                  "loc": {
                    "start": 5629,
                    "end": 5668
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
                        "start": 5680,
                        "end": 5684
                      }
                    },
                    "loc": {
                      "start": 5680,
                      "end": 5684
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
                            "start": 5698,
                            "end": 5706
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5695,
                          "end": 5706
                        }
                      }
                    ],
                    "loc": {
                      "start": 5685,
                      "end": 5712
                    }
                  },
                  "loc": {
                    "start": 5673,
                    "end": 5712
                  }
                }
              ],
              "loc": {
                "start": 5623,
                "end": 5714
              }
            },
            "loc": {
              "start": 5617,
              "end": 5714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5715,
                "end": 5726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5715,
              "end": 5726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5727,
                "end": 5741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5727,
              "end": 5741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5742,
                "end": 5747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5742,
              "end": 5747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5748,
                "end": 5757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5748,
              "end": 5757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5758,
                "end": 5762
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
                      "start": 5772,
                      "end": 5780
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5769,
                    "end": 5780
                  }
                }
              ],
              "loc": {
                "start": 5763,
                "end": 5782
              }
            },
            "loc": {
              "start": 5758,
              "end": 5782
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5783,
                "end": 5797
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5783,
              "end": 5797
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5798,
                "end": 5803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5798,
              "end": 5803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5804,
                "end": 5807
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
                      "start": 5814,
                      "end": 5823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5814,
                    "end": 5823
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5828,
                      "end": 5839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5828,
                    "end": 5839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 5844,
                      "end": 5855
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5844,
                    "end": 5855
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5860,
                      "end": 5869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5860,
                    "end": 5869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5874,
                      "end": 5881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5874,
                    "end": 5881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5886,
                      "end": 5894
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5886,
                    "end": 5894
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5899,
                      "end": 5911
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5899,
                    "end": 5911
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5916,
                      "end": 5924
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5916,
                    "end": 5924
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5929,
                      "end": 5937
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5929,
                    "end": 5937
                  }
                }
              ],
              "loc": {
                "start": 5808,
                "end": 5939
              }
            },
            "loc": {
              "start": 5804,
              "end": 5939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5940,
                "end": 5948
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
                      "start": 5955,
                      "end": 5957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5955,
                    "end": 5957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5962,
                      "end": 5972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5962,
                    "end": 5972
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5977,
                      "end": 5987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5977,
                    "end": 5987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5992,
                      "end": 6002
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5992,
                    "end": 6002
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6007,
                      "end": 6013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6007,
                    "end": 6013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6018,
                      "end": 6026
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6018,
                    "end": 6026
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6031,
                      "end": 6040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6031,
                    "end": 6040
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6045,
                      "end": 6052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6045,
                    "end": 6052
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6057,
                      "end": 6069
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6057,
                    "end": 6069
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6074,
                      "end": 6079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6074,
                    "end": 6079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6084,
                      "end": 6087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6084,
                    "end": 6087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6092,
                      "end": 6104
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6092,
                    "end": 6104
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6109,
                      "end": 6121
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6109,
                    "end": 6121
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6126,
                      "end": 6139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6126,
                    "end": 6139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6144,
                      "end": 6166
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6144,
                    "end": 6166
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6171,
                      "end": 6181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6171,
                    "end": 6181
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6186,
                      "end": 6198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6186,
                    "end": 6198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6203,
                      "end": 6206
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
                            "start": 6217,
                            "end": 6227
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6217,
                          "end": 6227
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6236,
                            "end": 6243
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6236,
                          "end": 6243
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
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
                          "value": "canReport",
                          "loc": {
                            "start": 6270,
                            "end": 6279
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6270,
                          "end": 6279
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6288,
                            "end": 6297
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6288,
                          "end": 6297
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6306,
                            "end": 6312
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6306,
                          "end": 6312
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6321,
                            "end": 6328
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6321,
                          "end": 6328
                        }
                      }
                    ],
                    "loc": {
                      "start": 6207,
                      "end": 6334
                    }
                  },
                  "loc": {
                    "start": 6203,
                    "end": 6334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6339,
                      "end": 6351
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
                            "start": 6362,
                            "end": 6364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6362,
                          "end": 6364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6373,
                            "end": 6381
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6373,
                          "end": 6381
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6390,
                            "end": 6401
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6390,
                          "end": 6401
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6410,
                            "end": 6422
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6410,
                          "end": 6422
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6431,
                            "end": 6435
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6431,
                          "end": 6435
                        }
                      }
                    ],
                    "loc": {
                      "start": 6352,
                      "end": 6441
                    }
                  },
                  "loc": {
                    "start": 6339,
                    "end": 6441
                  }
                }
              ],
              "loc": {
                "start": 5949,
                "end": 6443
              }
            },
            "loc": {
              "start": 5940,
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
              "value": "id",
              "loc": {
                "start": 7458,
                "end": 7460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7458,
              "end": 7460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7461,
                "end": 7471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7461,
              "end": 7471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7472,
                "end": 7482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7472,
              "end": 7482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7483,
                "end": 7494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7483,
              "end": 7494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7495,
                "end": 7501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7495,
              "end": 7501
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7502,
                "end": 7507
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7502,
              "end": 7507
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7508,
                "end": 7528
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7508,
              "end": 7528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7529,
                "end": 7533
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7529,
              "end": 7533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7534,
                "end": 7546
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7534,
              "end": 7546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7547,
                "end": 7556
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7547,
              "end": 7556
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7557,
                "end": 7577
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7557,
              "end": 7577
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7578,
                "end": 7581
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
                      "start": 7588,
                      "end": 7597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7588,
                    "end": 7597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7602,
                      "end": 7611
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7602,
                    "end": 7611
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7616,
                      "end": 7625
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7616,
                    "end": 7625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7630,
                      "end": 7642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7630,
                    "end": 7642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7647,
                      "end": 7655
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7647,
                    "end": 7655
                  }
                }
              ],
              "loc": {
                "start": 7582,
                "end": 7657
              }
            },
            "loc": {
              "start": 7578,
              "end": 7657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7658,
                "end": 7670
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
                      "start": 7677,
                      "end": 7679
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7677,
                    "end": 7679
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7684,
                      "end": 7692
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7684,
                    "end": 7692
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7697,
                      "end": 7700
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7697,
                    "end": 7700
                  }
                }
              ],
              "loc": {
                "start": 7671,
                "end": 7702
              }
            },
            "loc": {
              "start": 7658,
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
