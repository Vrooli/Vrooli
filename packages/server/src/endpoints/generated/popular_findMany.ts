export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 7892,
          "end": 7900
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7901,
              "end": 7906
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7909,
                "end": 7914
              }
            },
            "loc": {
              "start": 7908,
              "end": 7914
            }
          },
          "loc": {
            "start": 7901,
            "end": 7914
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
                "start": 7922,
                "end": 7927
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
                      "start": 7938,
                      "end": 7944
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7938,
                    "end": 7944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 7953,
                      "end": 7957
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
                              "start": 7979,
                              "end": 7982
                            }
                          },
                          "loc": {
                            "start": 7979,
                            "end": 7982
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
                                  "start": 8004,
                                  "end": 8012
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8001,
                                "end": 8012
                              }
                            }
                          ],
                          "loc": {
                            "start": 7983,
                            "end": 8026
                          }
                        },
                        "loc": {
                          "start": 7972,
                          "end": 8026
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
                              "start": 8046,
                              "end": 8050
                            }
                          },
                          "loc": {
                            "start": 8046,
                            "end": 8050
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
                                  "start": 8072,
                                  "end": 8081
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8069,
                                "end": 8081
                              }
                            }
                          ],
                          "loc": {
                            "start": 8051,
                            "end": 8095
                          }
                        },
                        "loc": {
                          "start": 8039,
                          "end": 8095
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
                              "start": 8115,
                              "end": 8119
                            }
                          },
                          "loc": {
                            "start": 8115,
                            "end": 8119
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
                                  "start": 8141,
                                  "end": 8150
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8138,
                                "end": 8150
                              }
                            }
                          ],
                          "loc": {
                            "start": 8120,
                            "end": 8164
                          }
                        },
                        "loc": {
                          "start": 8108,
                          "end": 8164
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
                              "start": 8184,
                              "end": 8191
                            }
                          },
                          "loc": {
                            "start": 8184,
                            "end": 8191
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
                                  "start": 8213,
                                  "end": 8225
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8210,
                                "end": 8225
                              }
                            }
                          ],
                          "loc": {
                            "start": 8192,
                            "end": 8239
                          }
                        },
                        "loc": {
                          "start": 8177,
                          "end": 8239
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
                              "start": 8259,
                              "end": 8267
                            }
                          },
                          "loc": {
                            "start": 8259,
                            "end": 8267
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
                                  "start": 8289,
                                  "end": 8302
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8286,
                                "end": 8302
                              }
                            }
                          ],
                          "loc": {
                            "start": 8268,
                            "end": 8316
                          }
                        },
                        "loc": {
                          "start": 8252,
                          "end": 8316
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
                              "start": 8336,
                              "end": 8343
                            }
                          },
                          "loc": {
                            "start": 8336,
                            "end": 8343
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
                                  "start": 8365,
                                  "end": 8377
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8362,
                                "end": 8377
                              }
                            }
                          ],
                          "loc": {
                            "start": 8344,
                            "end": 8391
                          }
                        },
                        "loc": {
                          "start": 8329,
                          "end": 8391
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
                              "start": 8411,
                              "end": 8419
                            }
                          },
                          "loc": {
                            "start": 8411,
                            "end": 8419
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
                                  "start": 8441,
                                  "end": 8454
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8438,
                                "end": 8454
                              }
                            }
                          ],
                          "loc": {
                            "start": 8420,
                            "end": 8468
                          }
                        },
                        "loc": {
                          "start": 8404,
                          "end": 8468
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
                              "start": 8488,
                              "end": 8492
                            }
                          },
                          "loc": {
                            "start": 8488,
                            "end": 8492
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
                                  "start": 8514,
                                  "end": 8523
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8511,
                                "end": 8523
                              }
                            }
                          ],
                          "loc": {
                            "start": 8493,
                            "end": 8537
                          }
                        },
                        "loc": {
                          "start": 8481,
                          "end": 8537
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
                              "start": 8557,
                              "end": 8561
                            }
                          },
                          "loc": {
                            "start": 8557,
                            "end": 8561
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
                                  "start": 8583,
                                  "end": 8592
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8580,
                                "end": 8592
                              }
                            }
                          ],
                          "loc": {
                            "start": 8562,
                            "end": 8606
                          }
                        },
                        "loc": {
                          "start": 8550,
                          "end": 8606
                        }
                      }
                    ],
                    "loc": {
                      "start": 7958,
                      "end": 8616
                    }
                  },
                  "loc": {
                    "start": 7953,
                    "end": 8616
                  }
                }
              ],
              "loc": {
                "start": 7928,
                "end": 8622
              }
            },
            "loc": {
              "start": 7922,
              "end": 8622
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8627,
                "end": 8635
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
                      "start": 8646,
                      "end": 8657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8646,
                    "end": 8657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8666,
                      "end": 8678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8666,
                    "end": 8678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorCode",
                    "loc": {
                      "start": 8687,
                      "end": 8700
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8687,
                    "end": 8700
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8709,
                      "end": 8722
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8709,
                    "end": 8722
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8731,
                      "end": 8747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8731,
                    "end": 8747
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8756,
                      "end": 8773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8756,
                    "end": 8773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8782,
                      "end": 8798
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8782,
                    "end": 8798
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 8807,
                      "end": 8824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8807,
                    "end": 8824
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorTeam",
                    "loc": {
                      "start": 8833,
                      "end": 8846
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8833,
                    "end": 8846
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 8855,
                      "end": 8868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8855,
                    "end": 8868
                  }
                }
              ],
              "loc": {
                "start": 8636,
                "end": 8874
              }
            },
            "loc": {
              "start": 8627,
              "end": 8874
            }
          }
        ],
        "loc": {
          "start": 7916,
          "end": 8878
        }
      },
      "loc": {
        "start": 7892,
        "end": 8878
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
              "value": "schemaLanguage",
              "loc": {
                "start": 761,
                "end": 775
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 775
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 780,
                "end": 792
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
                      "start": 803,
                      "end": 805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 803,
                    "end": 805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 814,
                      "end": 822
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 814,
                    "end": 822
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 831,
                      "end": 838
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 831,
                    "end": 838
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 847,
                      "end": 851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 847,
                    "end": 851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 860,
                      "end": 867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 860,
                    "end": 867
                  }
                }
              ],
              "loc": {
                "start": 793,
                "end": 873
              }
            },
            "loc": {
              "start": 780,
              "end": 873
            }
          }
        ],
        "loc": {
          "start": 436,
          "end": 875
        }
      },
      "loc": {
        "start": 427,
        "end": 875
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 904,
          "end": 906
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 906
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 907,
          "end": 916
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 907,
        "end": 916
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 948,
          "end": 950
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 948,
        "end": 950
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 951,
          "end": 961
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 951,
        "end": 961
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 962,
          "end": 972
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 962,
        "end": 972
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 973,
          "end": 982
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 973,
        "end": 982
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 983,
          "end": 994
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 983,
        "end": 994
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 995,
          "end": 1001
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
                "start": 1011,
                "end": 1021
              }
            },
            "directives": [],
            "loc": {
              "start": 1008,
              "end": 1021
            }
          }
        ],
        "loc": {
          "start": 1002,
          "end": 1023
        }
      },
      "loc": {
        "start": 995,
        "end": 1023
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1024,
          "end": 1029
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
                  "start": 1043,
                  "end": 1047
                }
              },
              "loc": {
                "start": 1043,
                "end": 1047
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
                      "start": 1061,
                      "end": 1069
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1058,
                    "end": 1069
                  }
                }
              ],
              "loc": {
                "start": 1048,
                "end": 1075
              }
            },
            "loc": {
              "start": 1036,
              "end": 1075
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
                  "start": 1087,
                  "end": 1091
                }
              },
              "loc": {
                "start": 1087,
                "end": 1091
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
                      "start": 1105,
                      "end": 1113
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1102,
                    "end": 1113
                  }
                }
              ],
              "loc": {
                "start": 1092,
                "end": 1119
              }
            },
            "loc": {
              "start": 1080,
              "end": 1119
            }
          }
        ],
        "loc": {
          "start": 1030,
          "end": 1121
        }
      },
      "loc": {
        "start": 1024,
        "end": 1121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1122,
          "end": 1133
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1122,
        "end": 1133
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1134,
          "end": 1148
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1134,
        "end": 1148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1149,
          "end": 1154
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1149,
        "end": 1154
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1155,
          "end": 1164
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1155,
        "end": 1164
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1165,
          "end": 1169
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
                "start": 1179,
                "end": 1187
              }
            },
            "directives": [],
            "loc": {
              "start": 1176,
              "end": 1187
            }
          }
        ],
        "loc": {
          "start": 1170,
          "end": 1189
        }
      },
      "loc": {
        "start": 1165,
        "end": 1189
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1190,
          "end": 1204
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1190,
        "end": 1204
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1205,
          "end": 1210
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1205,
        "end": 1210
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1211,
          "end": 1214
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
                "start": 1221,
                "end": 1230
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1221,
              "end": 1230
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1235,
                "end": 1246
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1235,
              "end": 1246
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1251,
                "end": 1262
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1251,
              "end": 1262
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1267,
                "end": 1276
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1267,
              "end": 1276
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1281,
                "end": 1288
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1281,
              "end": 1288
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1293,
                "end": 1301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1293,
              "end": 1301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1306,
                "end": 1318
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1306,
              "end": 1318
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1323,
                "end": 1331
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1323,
              "end": 1331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1336,
                "end": 1344
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1336,
              "end": 1344
            }
          }
        ],
        "loc": {
          "start": 1215,
          "end": 1346
        }
      },
      "loc": {
        "start": 1211,
        "end": 1346
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1347,
          "end": 1355
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
                "start": 1362,
                "end": 1364
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1362,
              "end": 1364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1369,
                "end": 1379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1369,
              "end": 1379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1384,
                "end": 1394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1384,
              "end": 1394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1399,
                "end": 1409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1399,
              "end": 1409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1414,
                "end": 1423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1414,
              "end": 1423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1428,
                "end": 1436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1428,
              "end": 1436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1441,
                "end": 1450
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1441,
              "end": 1450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeLanguage",
              "loc": {
                "start": 1455,
                "end": 1467
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1455,
              "end": 1467
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeType",
              "loc": {
                "start": 1472,
                "end": 1480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1472,
              "end": 1480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 1485,
                "end": 1492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1485,
              "end": 1492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1497,
                "end": 1509
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1497,
              "end": 1509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1514,
                "end": 1526
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1514,
              "end": 1526
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "calledByRoutineVersionsCount",
              "loc": {
                "start": 1531,
                "end": 1559
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1531,
              "end": 1559
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1564,
                "end": 1577
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1564,
              "end": 1577
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1582,
                "end": 1604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1582,
              "end": 1604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1609,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1609,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1624,
                "end": 1636
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1624,
              "end": 1636
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1641,
                "end": 1644
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
                      "start": 1655,
                      "end": 1665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1655,
                    "end": 1665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1674,
                      "end": 1681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1674,
                    "end": 1681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1690,
                      "end": 1699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1690,
                    "end": 1699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1708,
                      "end": 1717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1708,
                    "end": 1717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1726,
                      "end": 1735
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1726,
                    "end": 1735
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1744,
                      "end": 1750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1744,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1759,
                      "end": 1766
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1759,
                    "end": 1766
                  }
                }
              ],
              "loc": {
                "start": 1645,
                "end": 1772
              }
            },
            "loc": {
              "start": 1641,
              "end": 1772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1777,
                "end": 1789
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
                      "start": 1800,
                      "end": 1802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1800,
                    "end": 1802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1811,
                      "end": 1819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1811,
                    "end": 1819
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1828,
                      "end": 1839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1828,
                    "end": 1839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 1848,
                      "end": 1860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1848,
                    "end": 1860
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1869,
                      "end": 1873
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1869,
                    "end": 1873
                  }
                }
              ],
              "loc": {
                "start": 1790,
                "end": 1879
              }
            },
            "loc": {
              "start": 1777,
              "end": 1879
            }
          }
        ],
        "loc": {
          "start": 1356,
          "end": 1881
        }
      },
      "loc": {
        "start": 1347,
        "end": 1881
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1912,
          "end": 1914
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1912,
        "end": 1914
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1915,
          "end": 1924
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1915,
        "end": 1924
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1958,
          "end": 1960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1958,
        "end": 1960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1961,
          "end": 1971
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1961,
        "end": 1971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1972,
          "end": 1982
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1972,
        "end": 1982
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1983,
          "end": 1988
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1983,
        "end": 1988
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1989,
          "end": 1994
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1989,
        "end": 1994
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1995,
          "end": 2000
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
                  "start": 2014,
                  "end": 2018
                }
              },
              "loc": {
                "start": 2014,
                "end": 2018
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
                      "start": 2032,
                      "end": 2040
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2029,
                    "end": 2040
                  }
                }
              ],
              "loc": {
                "start": 2019,
                "end": 2046
              }
            },
            "loc": {
              "start": 2007,
              "end": 2046
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
                  "start": 2058,
                  "end": 2062
                }
              },
              "loc": {
                "start": 2058,
                "end": 2062
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
                      "start": 2076,
                      "end": 2084
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2073,
                    "end": 2084
                  }
                }
              ],
              "loc": {
                "start": 2063,
                "end": 2090
              }
            },
            "loc": {
              "start": 2051,
              "end": 2090
            }
          }
        ],
        "loc": {
          "start": 2001,
          "end": 2092
        }
      },
      "loc": {
        "start": 1995,
        "end": 2092
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2093,
          "end": 2096
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
                "start": 2103,
                "end": 2112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2103,
              "end": 2112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2117,
                "end": 2126
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2117,
              "end": 2126
            }
          }
        ],
        "loc": {
          "start": 2097,
          "end": 2128
        }
      },
      "loc": {
        "start": 2093,
        "end": 2128
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2160,
          "end": 2162
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2160,
        "end": 2162
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2163,
          "end": 2173
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2163,
        "end": 2173
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2174,
          "end": 2184
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2174,
        "end": 2184
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2185,
          "end": 2194
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2185,
        "end": 2194
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 2195,
          "end": 2206
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2195,
        "end": 2206
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2207,
          "end": 2213
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
                "start": 2223,
                "end": 2233
              }
            },
            "directives": [],
            "loc": {
              "start": 2220,
              "end": 2233
            }
          }
        ],
        "loc": {
          "start": 2214,
          "end": 2235
        }
      },
      "loc": {
        "start": 2207,
        "end": 2235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 2236,
          "end": 2241
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
                  "start": 2255,
                  "end": 2259
                }
              },
              "loc": {
                "start": 2255,
                "end": 2259
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
                      "start": 2273,
                      "end": 2281
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2270,
                    "end": 2281
                  }
                }
              ],
              "loc": {
                "start": 2260,
                "end": 2287
              }
            },
            "loc": {
              "start": 2248,
              "end": 2287
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
                  "start": 2299,
                  "end": 2303
                }
              },
              "loc": {
                "start": 2299,
                "end": 2303
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
                      "start": 2317,
                      "end": 2325
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2314,
                    "end": 2325
                  }
                }
              ],
              "loc": {
                "start": 2304,
                "end": 2331
              }
            },
            "loc": {
              "start": 2292,
              "end": 2331
            }
          }
        ],
        "loc": {
          "start": 2242,
          "end": 2333
        }
      },
      "loc": {
        "start": 2236,
        "end": 2333
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2334,
          "end": 2345
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2334,
        "end": 2345
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2346,
          "end": 2360
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2346,
        "end": 2360
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2361,
          "end": 2366
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2361,
        "end": 2366
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2367,
          "end": 2376
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2367,
        "end": 2376
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2377,
          "end": 2381
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
                "start": 2391,
                "end": 2399
              }
            },
            "directives": [],
            "loc": {
              "start": 2388,
              "end": 2399
            }
          }
        ],
        "loc": {
          "start": 2382,
          "end": 2401
        }
      },
      "loc": {
        "start": 2377,
        "end": 2401
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2402,
          "end": 2416
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2402,
        "end": 2416
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2417,
          "end": 2422
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2417,
        "end": 2422
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2423,
          "end": 2426
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
                "start": 2433,
                "end": 2442
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2433,
              "end": 2442
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2447,
                "end": 2458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2447,
              "end": 2458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2463,
                "end": 2474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2463,
              "end": 2474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2479,
                "end": 2488
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2479,
              "end": 2488
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2493,
                "end": 2500
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2493,
              "end": 2500
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2505,
                "end": 2513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2505,
              "end": 2513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2518,
                "end": 2530
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2518,
              "end": 2530
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2535,
                "end": 2543
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2535,
              "end": 2543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2548,
                "end": 2556
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2548,
              "end": 2556
            }
          }
        ],
        "loc": {
          "start": 2427,
          "end": 2558
        }
      },
      "loc": {
        "start": 2423,
        "end": 2558
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2559,
          "end": 2567
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
                "start": 2574,
                "end": 2576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2574,
              "end": 2576
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2581,
                "end": 2591
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2581,
              "end": 2591
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2596,
                "end": 2606
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2596,
              "end": 2606
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 2611,
                "end": 2619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2611,
              "end": 2619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2624,
                "end": 2633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2624,
              "end": 2633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2638,
                "end": 2650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2638,
              "end": 2650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 2655,
                "end": 2667
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2655,
              "end": 2667
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 2672,
                "end": 2684
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2672,
              "end": 2684
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2689,
                "end": 2692
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
                      "start": 2703,
                      "end": 2713
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2703,
                    "end": 2713
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 2722,
                      "end": 2729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2722,
                    "end": 2729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2738,
                      "end": 2747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2738,
                    "end": 2747
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2756,
                      "end": 2765
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2756,
                    "end": 2765
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2774,
                      "end": 2783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2774,
                    "end": 2783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 2792,
                      "end": 2798
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2792,
                    "end": 2798
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2807,
                      "end": 2814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2807,
                    "end": 2814
                  }
                }
              ],
              "loc": {
                "start": 2693,
                "end": 2820
              }
            },
            "loc": {
              "start": 2689,
              "end": 2820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2825,
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
                    "value": "id",
                    "loc": {
                      "start": 2848,
                      "end": 2850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2848,
                    "end": 2850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2859,
                      "end": 2867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2859,
                    "end": 2867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2876,
                      "end": 2887
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2876,
                    "end": 2887
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2896,
                      "end": 2900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2896,
                    "end": 2900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 2909,
                      "end": 2914
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
                            "start": 2929,
                            "end": 2931
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2929,
                          "end": 2931
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 2944,
                            "end": 2953
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2944,
                          "end": 2953
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 2966,
                            "end": 2970
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2966,
                          "end": 2970
                        }
                      }
                    ],
                    "loc": {
                      "start": 2915,
                      "end": 2980
                    }
                  },
                  "loc": {
                    "start": 2909,
                    "end": 2980
                  }
                }
              ],
              "loc": {
                "start": 2838,
                "end": 2986
              }
            },
            "loc": {
              "start": 2825,
              "end": 2986
            }
          }
        ],
        "loc": {
          "start": 2568,
          "end": 2988
        }
      },
      "loc": {
        "start": 2559,
        "end": 2988
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3019,
          "end": 3021
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3019,
        "end": 3021
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3022,
          "end": 3031
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3022,
        "end": 3031
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3069,
          "end": 3071
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3069,
        "end": 3071
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3072,
          "end": 3082
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3072,
        "end": 3082
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3083,
          "end": 3093
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3083,
        "end": 3093
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3094,
          "end": 3103
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3094,
        "end": 3103
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3104,
          "end": 3115
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3104,
        "end": 3115
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3116,
          "end": 3122
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
                "start": 3132,
                "end": 3142
              }
            },
            "directives": [],
            "loc": {
              "start": 3129,
              "end": 3142
            }
          }
        ],
        "loc": {
          "start": 3123,
          "end": 3144
        }
      },
      "loc": {
        "start": 3116,
        "end": 3144
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3145,
          "end": 3150
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
                  "start": 3164,
                  "end": 3168
                }
              },
              "loc": {
                "start": 3164,
                "end": 3168
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
                      "start": 3182,
                      "end": 3190
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3179,
                    "end": 3190
                  }
                }
              ],
              "loc": {
                "start": 3169,
                "end": 3196
              }
            },
            "loc": {
              "start": 3157,
              "end": 3196
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
                  "start": 3208,
                  "end": 3212
                }
              },
              "loc": {
                "start": 3208,
                "end": 3212
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
                      "start": 3226,
                      "end": 3234
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3223,
                    "end": 3234
                  }
                }
              ],
              "loc": {
                "start": 3213,
                "end": 3240
              }
            },
            "loc": {
              "start": 3201,
              "end": 3240
            }
          }
        ],
        "loc": {
          "start": 3151,
          "end": 3242
        }
      },
      "loc": {
        "start": 3145,
        "end": 3242
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3243,
          "end": 3254
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3243,
        "end": 3254
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3255,
          "end": 3269
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3255,
        "end": 3269
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3270,
          "end": 3275
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3270,
        "end": 3275
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3276,
          "end": 3285
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3276,
        "end": 3285
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3286,
          "end": 3290
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
                "start": 3300,
                "end": 3308
              }
            },
            "directives": [],
            "loc": {
              "start": 3297,
              "end": 3308
            }
          }
        ],
        "loc": {
          "start": 3291,
          "end": 3310
        }
      },
      "loc": {
        "start": 3286,
        "end": 3310
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3311,
          "end": 3325
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3311,
        "end": 3325
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3326,
          "end": 3331
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3326,
        "end": 3331
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3332,
          "end": 3335
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
                "start": 3342,
                "end": 3351
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3342,
              "end": 3351
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
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
              "value": "canTransfer",
              "loc": {
                "start": 3372,
                "end": 3383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3372,
              "end": 3383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3388,
                "end": 3397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3388,
              "end": 3397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3402,
                "end": 3409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3402,
              "end": 3409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3414,
                "end": 3422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3414,
              "end": 3422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3427,
                "end": 3439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3427,
              "end": 3439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3444,
                "end": 3452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3444,
              "end": 3452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3457,
                "end": 3465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3457,
              "end": 3465
            }
          }
        ],
        "loc": {
          "start": 3336,
          "end": 3467
        }
      },
      "loc": {
        "start": 3332,
        "end": 3467
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3468,
          "end": 3476
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
                "start": 3483,
                "end": 3485
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3483,
              "end": 3485
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3490,
                "end": 3500
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3490,
              "end": 3500
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3505,
                "end": 3515
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3505,
              "end": 3515
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3520,
                "end": 3536
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3520,
              "end": 3536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3541,
                "end": 3549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3541,
              "end": 3549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3554,
                "end": 3563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3554,
              "end": 3563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3568,
                "end": 3580
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3568,
              "end": 3580
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3585,
                "end": 3601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3585,
              "end": 3601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3606,
                "end": 3616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3606,
              "end": 3616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3621,
                "end": 3633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3621,
              "end": 3633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3638,
                "end": 3650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3638,
              "end": 3650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3655,
                "end": 3667
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
                      "start": 3678,
                      "end": 3680
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3678,
                    "end": 3680
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3689,
                      "end": 3697
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3689,
                    "end": 3697
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3706,
                      "end": 3717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3706,
                    "end": 3717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3726,
                      "end": 3730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3726,
                    "end": 3730
                  }
                }
              ],
              "loc": {
                "start": 3668,
                "end": 3736
              }
            },
            "loc": {
              "start": 3655,
              "end": 3736
            }
          }
        ],
        "loc": {
          "start": 3477,
          "end": 3738
        }
      },
      "loc": {
        "start": 3468,
        "end": 3738
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3775,
          "end": 3777
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3775,
        "end": 3777
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3778,
          "end": 3787
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3778,
        "end": 3787
      }
    },
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
        "value": "created_at",
        "loc": {
          "start": 3830,
          "end": 3840
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3830,
        "end": 3840
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3841,
          "end": 3851
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3841,
        "end": 3851
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3852,
          "end": 3861
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
                "start": 3868,
                "end": 3870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3868,
              "end": 3870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3875,
                "end": 3885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3875,
              "end": 3885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3890,
                "end": 3900
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3890,
              "end": 3900
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3905,
                "end": 3916
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3905,
              "end": 3916
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3921,
                "end": 3927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3921,
              "end": 3927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3932,
                "end": 3937
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3932,
              "end": 3937
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 3942,
                "end": 3962
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3942,
              "end": 3962
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3967,
                "end": 3971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3967,
              "end": 3971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3976,
                "end": 3988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3976,
              "end": 3988
            }
          }
        ],
        "loc": {
          "start": 3862,
          "end": 3990
        }
      },
      "loc": {
        "start": 3852,
        "end": 3990
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 3991,
          "end": 4008
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3991,
        "end": 4008
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4009,
          "end": 4018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4009,
        "end": 4018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4019,
          "end": 4024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4019,
        "end": 4024
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4025,
          "end": 4034
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4025,
        "end": 4034
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4035,
          "end": 4047
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4035,
        "end": 4047
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4048,
          "end": 4061
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4048,
        "end": 4061
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4062,
          "end": 4074
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4062,
        "end": 4074
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4075,
          "end": 4084
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
                  "start": 4098,
                  "end": 4101
                }
              },
              "loc": {
                "start": 4098,
                "end": 4101
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
                      "start": 4115,
                      "end": 4122
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4112,
                    "end": 4122
                  }
                }
              ],
              "loc": {
                "start": 4102,
                "end": 4128
              }
            },
            "loc": {
              "start": 4091,
              "end": 4128
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
                  "start": 4140,
                  "end": 4144
                }
              },
              "loc": {
                "start": 4140,
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
                    "value": "Code_nav",
                    "loc": {
                      "start": 4158,
                      "end": 4166
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4155,
                    "end": 4166
                  }
                }
              ],
              "loc": {
                "start": 4145,
                "end": 4172
              }
            },
            "loc": {
              "start": 4133,
              "end": 4172
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
                  "start": 4184,
                  "end": 4188
                }
              },
              "loc": {
                "start": 4184,
                "end": 4188
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
                      "start": 4202,
                      "end": 4210
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4199,
                    "end": 4210
                  }
                }
              ],
              "loc": {
                "start": 4189,
                "end": 4216
              }
            },
            "loc": {
              "start": 4177,
              "end": 4216
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
                  "start": 4228,
                  "end": 4235
                }
              },
              "loc": {
                "start": 4228,
                "end": 4235
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
                      "start": 4249,
                      "end": 4260
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4246,
                    "end": 4260
                  }
                }
              ],
              "loc": {
                "start": 4236,
                "end": 4266
              }
            },
            "loc": {
              "start": 4221,
              "end": 4266
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
                  "start": 4278,
                  "end": 4285
                }
              },
              "loc": {
                "start": 4278,
                "end": 4285
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
                      "start": 4299,
                      "end": 4310
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4296,
                    "end": 4310
                  }
                }
              ],
              "loc": {
                "start": 4286,
                "end": 4316
              }
            },
            "loc": {
              "start": 4271,
              "end": 4316
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
                  "start": 4328,
                  "end": 4336
                }
              },
              "loc": {
                "start": 4328,
                "end": 4336
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
                      "start": 4350,
                      "end": 4362
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4347,
                    "end": 4362
                  }
                }
              ],
              "loc": {
                "start": 4337,
                "end": 4368
              }
            },
            "loc": {
              "start": 4321,
              "end": 4368
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
                  "start": 4380,
                  "end": 4384
                }
              },
              "loc": {
                "start": 4380,
                "end": 4384
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
                      "start": 4398,
                      "end": 4406
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4395,
                    "end": 4406
                  }
                }
              ],
              "loc": {
                "start": 4385,
                "end": 4412
              }
            },
            "loc": {
              "start": 4373,
              "end": 4412
            }
          }
        ],
        "loc": {
          "start": 4085,
          "end": 4414
        }
      },
      "loc": {
        "start": 4075,
        "end": 4414
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4415,
          "end": 4419
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
                "start": 4429,
                "end": 4437
              }
            },
            "directives": [],
            "loc": {
              "start": 4426,
              "end": 4437
            }
          }
        ],
        "loc": {
          "start": 4420,
          "end": 4439
        }
      },
      "loc": {
        "start": 4415,
        "end": 4439
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4440,
          "end": 4443
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
                "start": 4450,
                "end": 4458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4450,
              "end": 4458
            }
          }
        ],
        "loc": {
          "start": 4444,
          "end": 4460
        }
      },
      "loc": {
        "start": 4440,
        "end": 4460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 4461,
          "end": 4473
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
                "start": 4480,
                "end": 4482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4480,
              "end": 4482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 4487,
                "end": 4495
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4487,
              "end": 4495
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 4500,
                "end": 4511
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4500,
              "end": 4511
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 4516,
                "end": 4520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4516,
              "end": 4520
            }
          }
        ],
        "loc": {
          "start": 4474,
          "end": 4522
        }
      },
      "loc": {
        "start": 4461,
        "end": 4522
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4560,
          "end": 4562
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4560,
        "end": 4562
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4563,
          "end": 4573
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4563,
        "end": 4573
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4574,
          "end": 4584
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4574,
        "end": 4584
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 4585,
          "end": 4595
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4585,
        "end": 4595
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4596,
          "end": 4605
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4596,
        "end": 4605
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 4606,
          "end": 4617
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4606,
        "end": 4617
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 4618,
          "end": 4624
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
                "start": 4634,
                "end": 4644
              }
            },
            "directives": [],
            "loc": {
              "start": 4631,
              "end": 4644
            }
          }
        ],
        "loc": {
          "start": 4625,
          "end": 4646
        }
      },
      "loc": {
        "start": 4618,
        "end": 4646
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 4647,
          "end": 4652
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
                  "start": 4666,
                  "end": 4670
                }
              },
              "loc": {
                "start": 4666,
                "end": 4670
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
                      "start": 4684,
                      "end": 4692
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4681,
                    "end": 4692
                  }
                }
              ],
              "loc": {
                "start": 4671,
                "end": 4698
              }
            },
            "loc": {
              "start": 4659,
              "end": 4698
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
                  "start": 4710,
                  "end": 4714
                }
              },
              "loc": {
                "start": 4710,
                "end": 4714
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
                      "start": 4728,
                      "end": 4736
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4725,
                    "end": 4736
                  }
                }
              ],
              "loc": {
                "start": 4715,
                "end": 4742
              }
            },
            "loc": {
              "start": 4703,
              "end": 4742
            }
          }
        ],
        "loc": {
          "start": 4653,
          "end": 4744
        }
      },
      "loc": {
        "start": 4647,
        "end": 4744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 4745,
          "end": 4756
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4745,
        "end": 4756
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 4757,
          "end": 4771
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4757,
        "end": 4771
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4772,
          "end": 4777
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4772,
        "end": 4777
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4778,
          "end": 4787
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4778,
        "end": 4787
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4788,
          "end": 4792
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
                "start": 4802,
                "end": 4810
              }
            },
            "directives": [],
            "loc": {
              "start": 4799,
              "end": 4810
            }
          }
        ],
        "loc": {
          "start": 4793,
          "end": 4812
        }
      },
      "loc": {
        "start": 4788,
        "end": 4812
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 4813,
          "end": 4827
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4813,
        "end": 4827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 4828,
          "end": 4833
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4828,
        "end": 4833
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4834,
          "end": 4837
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
                "start": 4844,
                "end": 4854
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4844,
              "end": 4854
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 4859,
                "end": 4868
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4859,
              "end": 4868
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 4873,
                "end": 4884
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4873,
              "end": 4884
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 4889,
                "end": 4898
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4889,
              "end": 4898
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 4903,
                "end": 4910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4903,
              "end": 4910
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 4915,
                "end": 4923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4915,
              "end": 4923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 4928,
                "end": 4940
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4928,
              "end": 4940
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 4945,
                "end": 4953
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4945,
              "end": 4953
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 4958,
                "end": 4966
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4958,
              "end": 4966
            }
          }
        ],
        "loc": {
          "start": 4838,
          "end": 4968
        }
      },
      "loc": {
        "start": 4834,
        "end": 4968
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4969,
          "end": 4977
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
                "start": 4984,
                "end": 4986
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4984,
              "end": 4986
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4991,
                "end": 5001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4991,
              "end": 5001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5006,
                "end": 5016
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5006,
              "end": 5016
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 5021,
                "end": 5032
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5021,
              "end": 5032
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 5037,
                "end": 5050
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5037,
              "end": 5050
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5055,
                "end": 5065
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5055,
              "end": 5065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 5070,
                "end": 5079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5070,
              "end": 5079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5084,
                "end": 5092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5084,
              "end": 5092
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5097,
                "end": 5106
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5097,
              "end": 5106
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineType",
              "loc": {
                "start": 5111,
                "end": 5122
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5111,
              "end": 5122
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 5127,
                "end": 5137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5127,
              "end": 5137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 5142,
                "end": 5154
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5142,
              "end": 5154
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 5159,
                "end": 5173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5159,
              "end": 5173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5178,
                "end": 5190
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5178,
              "end": 5190
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5195,
                "end": 5207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5195,
              "end": 5207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5212,
                "end": 5225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5212,
              "end": 5225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5230,
                "end": 5252
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5230,
              "end": 5252
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5257,
                "end": 5267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5257,
              "end": 5267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 5272,
                "end": 5283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5272,
              "end": 5283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 5288,
                "end": 5298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5288,
              "end": 5298
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 5303,
                "end": 5317
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5303,
              "end": 5317
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 5322,
                "end": 5334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5322,
              "end": 5334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5339,
                "end": 5351
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5339,
              "end": 5351
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 5356,
                "end": 5368
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
                      "start": 5379,
                      "end": 5381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5379,
                    "end": 5381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5390,
                      "end": 5398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5390,
                    "end": 5398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5407,
                      "end": 5418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5407,
                    "end": 5418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 5427,
                      "end": 5439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5427,
                    "end": 5439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5448,
                      "end": 5452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5448,
                    "end": 5452
                  }
                }
              ],
              "loc": {
                "start": 5369,
                "end": 5458
              }
            },
            "loc": {
              "start": 5356,
              "end": 5458
            }
          }
        ],
        "loc": {
          "start": 4978,
          "end": 5460
        }
      },
      "loc": {
        "start": 4969,
        "end": 5460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5497,
          "end": 5499
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5497,
        "end": 5499
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5500,
          "end": 5510
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5500,
        "end": 5510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5511,
          "end": 5520
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5511,
        "end": 5520
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5560,
          "end": 5562
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5560,
        "end": 5562
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5563,
          "end": 5573
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5563,
        "end": 5573
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5574,
          "end": 5584
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5574,
        "end": 5584
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5585,
          "end": 5594
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5585,
        "end": 5594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5595,
          "end": 5606
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5595,
        "end": 5606
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5607,
          "end": 5613
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
                "start": 5623,
                "end": 5633
              }
            },
            "directives": [],
            "loc": {
              "start": 5620,
              "end": 5633
            }
          }
        ],
        "loc": {
          "start": 5614,
          "end": 5635
        }
      },
      "loc": {
        "start": 5607,
        "end": 5635
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5636,
          "end": 5641
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
                  "start": 5655,
                  "end": 5659
                }
              },
              "loc": {
                "start": 5655,
                "end": 5659
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
                      "start": 5673,
                      "end": 5681
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5670,
                    "end": 5681
                  }
                }
              ],
              "loc": {
                "start": 5660,
                "end": 5687
              }
            },
            "loc": {
              "start": 5648,
              "end": 5687
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
                  "start": 5699,
                  "end": 5703
                }
              },
              "loc": {
                "start": 5699,
                "end": 5703
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
                      "start": 5717,
                      "end": 5725
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5714,
                    "end": 5725
                  }
                }
              ],
              "loc": {
                "start": 5704,
                "end": 5731
              }
            },
            "loc": {
              "start": 5692,
              "end": 5731
            }
          }
        ],
        "loc": {
          "start": 5642,
          "end": 5733
        }
      },
      "loc": {
        "start": 5636,
        "end": 5733
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5734,
          "end": 5745
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5734,
        "end": 5745
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5746,
          "end": 5760
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5746,
        "end": 5760
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5761,
          "end": 5766
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5761,
        "end": 5766
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5767,
          "end": 5776
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5767,
        "end": 5776
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5777,
          "end": 5781
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
                "start": 5791,
                "end": 5799
              }
            },
            "directives": [],
            "loc": {
              "start": 5788,
              "end": 5799
            }
          }
        ],
        "loc": {
          "start": 5782,
          "end": 5801
        }
      },
      "loc": {
        "start": 5777,
        "end": 5801
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5802,
          "end": 5816
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5802,
        "end": 5816
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5817,
          "end": 5822
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5817,
        "end": 5822
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5823,
          "end": 5826
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
                "start": 5833,
                "end": 5842
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5833,
              "end": 5842
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5847,
                "end": 5858
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5847,
              "end": 5858
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 5863,
                "end": 5874
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5863,
              "end": 5874
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5879,
                "end": 5888
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5879,
              "end": 5888
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5893,
                "end": 5900
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5893,
              "end": 5900
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5905,
                "end": 5913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5905,
              "end": 5913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5918,
                "end": 5930
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5918,
              "end": 5930
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5935,
                "end": 5943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5935,
              "end": 5943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5948,
                "end": 5956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5948,
              "end": 5956
            }
          }
        ],
        "loc": {
          "start": 5827,
          "end": 5958
        }
      },
      "loc": {
        "start": 5823,
        "end": 5958
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5959,
          "end": 5967
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
                "start": 5974,
                "end": 5976
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5974,
              "end": 5976
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5981,
                "end": 5991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5981,
              "end": 5991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5996,
                "end": 6006
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5996,
              "end": 6006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6011,
                "end": 6021
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6011,
              "end": 6021
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 6026,
                "end": 6032
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6026,
              "end": 6032
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6037,
                "end": 6045
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6037,
              "end": 6045
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6050,
                "end": 6059
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6050,
              "end": 6059
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6064,
                "end": 6071
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6064,
              "end": 6071
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 6076,
                "end": 6088
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6076,
              "end": 6088
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 6093,
                "end": 6098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6093,
              "end": 6098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 6103,
                "end": 6106
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6103,
              "end": 6106
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6111,
                "end": 6123
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6111,
              "end": 6123
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6128,
                "end": 6140
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6128,
              "end": 6140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6145,
                "end": 6158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6145,
              "end": 6158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6163,
                "end": 6185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6163,
              "end": 6185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6190,
                "end": 6200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6190,
              "end": 6200
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6205,
                "end": 6217
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6205,
              "end": 6217
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6222,
                "end": 6225
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
                      "start": 6236,
                      "end": 6246
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6236,
                    "end": 6246
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6255,
                      "end": 6262
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6255,
                    "end": 6262
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6271,
                      "end": 6280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6271,
                    "end": 6280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6289,
                      "end": 6298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6289,
                    "end": 6298
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6307,
                      "end": 6316
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6307,
                    "end": 6316
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6325,
                      "end": 6331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6325,
                    "end": 6331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6340,
                      "end": 6347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6340,
                    "end": 6347
                  }
                }
              ],
              "loc": {
                "start": 6226,
                "end": 6353
              }
            },
            "loc": {
              "start": 6222,
              "end": 6353
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6358,
                "end": 6370
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
                      "start": 6381,
                      "end": 6383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6381,
                    "end": 6383
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6392,
                      "end": 6400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6392,
                    "end": 6400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6409,
                      "end": 6420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6409,
                    "end": 6420
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6429,
                      "end": 6441
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6429,
                    "end": 6441
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6450,
                      "end": 6454
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6450,
                    "end": 6454
                  }
                }
              ],
              "loc": {
                "start": 6371,
                "end": 6460
              }
            },
            "loc": {
              "start": 6358,
              "end": 6460
            }
          }
        ],
        "loc": {
          "start": 5968,
          "end": 6462
        }
      },
      "loc": {
        "start": 5959,
        "end": 6462
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6501,
          "end": 6503
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6501,
        "end": 6503
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6504,
          "end": 6513
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6504,
        "end": 6513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6543,
          "end": 6545
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6543,
        "end": 6545
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6546,
          "end": 6556
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6546,
        "end": 6556
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6557,
          "end": 6560
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6557,
        "end": 6560
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6561,
          "end": 6570
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6561,
        "end": 6570
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6571,
          "end": 6583
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
                "start": 6590,
                "end": 6592
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6590,
              "end": 6592
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6597,
                "end": 6605
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6597,
              "end": 6605
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6610,
                "end": 6621
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6610,
              "end": 6621
            }
          }
        ],
        "loc": {
          "start": 6584,
          "end": 6623
        }
      },
      "loc": {
        "start": 6571,
        "end": 6623
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6624,
          "end": 6627
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
                "start": 6634,
                "end": 6639
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6634,
              "end": 6639
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6644,
                "end": 6656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6644,
              "end": 6656
            }
          }
        ],
        "loc": {
          "start": 6628,
          "end": 6658
        }
      },
      "loc": {
        "start": 6624,
        "end": 6658
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6690,
          "end": 6692
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6690,
        "end": 6692
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 6693,
          "end": 6704
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6693,
        "end": 6704
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 6705,
          "end": 6711
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6705,
        "end": 6711
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6712,
          "end": 6722
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6712,
        "end": 6722
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6723,
          "end": 6733
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6723,
        "end": 6733
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 6734,
          "end": 6752
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6734,
        "end": 6752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6753,
          "end": 6762
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6753,
        "end": 6762
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 6763,
          "end": 6776
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6763,
        "end": 6776
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 6777,
          "end": 6789
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6777,
        "end": 6789
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 6790,
          "end": 6802
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6790,
        "end": 6802
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 6803,
          "end": 6815
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6803,
        "end": 6815
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6816,
          "end": 6825
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6816,
        "end": 6825
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6826,
          "end": 6830
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
                "start": 6840,
                "end": 6848
              }
            },
            "directives": [],
            "loc": {
              "start": 6837,
              "end": 6848
            }
          }
        ],
        "loc": {
          "start": 6831,
          "end": 6850
        }
      },
      "loc": {
        "start": 6826,
        "end": 6850
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6851,
          "end": 6863
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
                "start": 6870,
                "end": 6872
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6870,
              "end": 6872
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6877,
                "end": 6885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6877,
              "end": 6885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 6890,
                "end": 6893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6890,
              "end": 6893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6898,
                "end": 6902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6898,
              "end": 6902
            }
          }
        ],
        "loc": {
          "start": 6864,
          "end": 6904
        }
      },
      "loc": {
        "start": 6851,
        "end": 6904
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6905,
          "end": 6908
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
                "start": 6915,
                "end": 6928
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6915,
              "end": 6928
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6933,
                "end": 6942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6933,
              "end": 6942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6947,
                "end": 6958
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6947,
              "end": 6958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 6963,
                "end": 6972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6963,
              "end": 6972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6977,
                "end": 6986
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6977,
              "end": 6986
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6991,
                "end": 6998
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6991,
              "end": 6998
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7003,
                "end": 7015
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7003,
              "end": 7015
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7020,
                "end": 7028
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7020,
              "end": 7028
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7033,
                "end": 7047
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
                      "start": 7058,
                      "end": 7060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7058,
                    "end": 7060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
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
                    "value": "updated_at",
                    "loc": {
                      "start": 7088,
                      "end": 7098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7088,
                    "end": 7098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7107,
                      "end": 7114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7107,
                    "end": 7114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7123,
                      "end": 7134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7123,
                    "end": 7134
                  }
                }
              ],
              "loc": {
                "start": 7048,
                "end": 7140
              }
            },
            "loc": {
              "start": 7033,
              "end": 7140
            }
          }
        ],
        "loc": {
          "start": 6909,
          "end": 7142
        }
      },
      "loc": {
        "start": 6905,
        "end": 7142
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7173,
          "end": 7175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7173,
        "end": 7175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7176,
          "end": 7187
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7176,
        "end": 7187
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7188,
          "end": 7194
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7188,
        "end": 7194
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7195,
          "end": 7207
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7195,
        "end": 7207
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7208,
          "end": 7211
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
                "start": 7218,
                "end": 7231
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7218,
              "end": 7231
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7236,
                "end": 7245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7236,
              "end": 7245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7250,
                "end": 7261
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7250,
              "end": 7261
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7266,
                "end": 7275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7266,
              "end": 7275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7280,
                "end": 7289
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7280,
              "end": 7289
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7294,
                "end": 7301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7294,
              "end": 7301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7306,
                "end": 7318
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7306,
              "end": 7318
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7323,
                "end": 7331
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7323,
              "end": 7331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7336,
                "end": 7350
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
                      "start": 7361,
                      "end": 7363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7361,
                    "end": 7363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
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
                    "value": "updated_at",
                    "loc": {
                      "start": 7391,
                      "end": 7401
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7391,
                    "end": 7401
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7410,
                      "end": 7417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7410,
                    "end": 7417
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7426,
                      "end": 7437
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7426,
                    "end": 7437
                  }
                }
              ],
              "loc": {
                "start": 7351,
                "end": 7443
              }
            },
            "loc": {
              "start": 7336,
              "end": 7443
            }
          }
        ],
        "loc": {
          "start": 7212,
          "end": 7445
        }
      },
      "loc": {
        "start": 7208,
        "end": 7445
      }
    },
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
        "value": "created_at",
        "loc": {
          "start": 7480,
          "end": 7490
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7480,
        "end": 7490
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7491,
          "end": 7501
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7491,
        "end": 7501
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7502,
          "end": 7513
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7502,
        "end": 7513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7514,
          "end": 7520
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7514,
        "end": 7520
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7521,
          "end": 7526
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7521,
        "end": 7526
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7527,
          "end": 7547
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7527,
        "end": 7547
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7548,
          "end": 7552
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7548,
        "end": 7552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7553,
          "end": 7565
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7553,
        "end": 7565
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7566,
          "end": 7575
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7566,
        "end": 7575
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7576,
          "end": 7596
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7576,
        "end": 7596
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7597,
          "end": 7600
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
                "start": 7607,
                "end": 7616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7607,
              "end": 7616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7621,
                "end": 7630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7621,
              "end": 7630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7635,
                "end": 7644
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7635,
              "end": 7644
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7649,
                "end": 7661
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7649,
              "end": 7661
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7666,
                "end": 7674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7666,
              "end": 7674
            }
          }
        ],
        "loc": {
          "start": 7601,
          "end": 7676
        }
      },
      "loc": {
        "start": 7597,
        "end": 7676
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7677,
          "end": 7689
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
                "start": 7696,
                "end": 7698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7696,
              "end": 7698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7703,
                "end": 7711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7703,
              "end": 7711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7716,
                "end": 7719
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7716,
              "end": 7719
            }
          }
        ],
        "loc": {
          "start": 7690,
          "end": 7721
        }
      },
      "loc": {
        "start": 7677,
        "end": 7721
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7752,
          "end": 7754
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7752,
        "end": 7754
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7755,
          "end": 7765
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7755,
        "end": 7765
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7766,
          "end": 7776
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7766,
        "end": 7776
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7777,
          "end": 7788
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7777,
        "end": 7788
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7789,
          "end": 7795
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7789,
        "end": 7795
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7796,
          "end": 7801
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7796,
        "end": 7801
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7802,
          "end": 7822
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7802,
        "end": 7822
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7823,
          "end": 7827
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7823,
        "end": 7827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7828,
          "end": 7840
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7828,
        "end": 7840
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
                    "value": "schemaLanguage",
                    "loc": {
                      "start": 761,
                      "end": 775
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 775
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 780,
                      "end": 792
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
                            "start": 803,
                            "end": 805
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 803,
                          "end": 805
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 814,
                            "end": 822
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 814,
                          "end": 822
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 831,
                            "end": 838
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 831,
                          "end": 838
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 847,
                            "end": 851
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 847,
                          "end": 851
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 860,
                            "end": 867
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 860,
                          "end": 867
                        }
                      }
                    ],
                    "loc": {
                      "start": 793,
                      "end": 873
                    }
                  },
                  "loc": {
                    "start": 780,
                    "end": 873
                  }
                }
              ],
              "loc": {
                "start": 436,
                "end": 875
              }
            },
            "loc": {
              "start": 427,
              "end": 875
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 877
        }
      },
      "loc": {
        "start": 1,
        "end": 877
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 887,
          "end": 894
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 898,
            "end": 901
          }
        },
        "loc": {
          "start": 898,
          "end": 901
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
                "start": 904,
                "end": 906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 907,
                "end": 916
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 907,
              "end": 916
            }
          }
        ],
        "loc": {
          "start": 902,
          "end": 918
        }
      },
      "loc": {
        "start": 878,
        "end": 918
      }
    },
    "Code_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_list",
        "loc": {
          "start": 928,
          "end": 937
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 941,
            "end": 945
          }
        },
        "loc": {
          "start": 941,
          "end": 945
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
                "start": 948,
                "end": 950
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 948,
              "end": 950
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 951,
                "end": 961
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 951,
              "end": 961
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 962,
                "end": 972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 962,
              "end": 972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 973,
                "end": 982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 973,
              "end": 982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 983,
                "end": 994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 983,
              "end": 994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 995,
                "end": 1001
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
                      "start": 1011,
                      "end": 1021
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1008,
                    "end": 1021
                  }
                }
              ],
              "loc": {
                "start": 1002,
                "end": 1023
              }
            },
            "loc": {
              "start": 995,
              "end": 1023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1024,
                "end": 1029
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
                        "start": 1043,
                        "end": 1047
                      }
                    },
                    "loc": {
                      "start": 1043,
                      "end": 1047
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
                            "start": 1061,
                            "end": 1069
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1058,
                          "end": 1069
                        }
                      }
                    ],
                    "loc": {
                      "start": 1048,
                      "end": 1075
                    }
                  },
                  "loc": {
                    "start": 1036,
                    "end": 1075
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
                        "start": 1087,
                        "end": 1091
                      }
                    },
                    "loc": {
                      "start": 1087,
                      "end": 1091
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
                            "start": 1105,
                            "end": 1113
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1102,
                          "end": 1113
                        }
                      }
                    ],
                    "loc": {
                      "start": 1092,
                      "end": 1119
                    }
                  },
                  "loc": {
                    "start": 1080,
                    "end": 1119
                  }
                }
              ],
              "loc": {
                "start": 1030,
                "end": 1121
              }
            },
            "loc": {
              "start": 1024,
              "end": 1121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1122,
                "end": 1133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1134,
                "end": 1148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1134,
              "end": 1148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1149,
                "end": 1154
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1149,
              "end": 1154
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1155,
                "end": 1164
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1155,
              "end": 1164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1165,
                "end": 1169
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
                      "start": 1179,
                      "end": 1187
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1176,
                    "end": 1187
                  }
                }
              ],
              "loc": {
                "start": 1170,
                "end": 1189
              }
            },
            "loc": {
              "start": 1165,
              "end": 1189
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1190,
                "end": 1204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1190,
              "end": 1204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1205,
                "end": 1210
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1205,
              "end": 1210
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1211,
                "end": 1214
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
                      "start": 1221,
                      "end": 1230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1221,
                    "end": 1230
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1235,
                      "end": 1246
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1235,
                    "end": 1246
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1251,
                      "end": 1262
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1251,
                    "end": 1262
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1267,
                      "end": 1276
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1267,
                    "end": 1276
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1281,
                      "end": 1288
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1281,
                    "end": 1288
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1293,
                      "end": 1301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1293,
                    "end": 1301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1306,
                      "end": 1318
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1306,
                    "end": 1318
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1323,
                      "end": 1331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1323,
                    "end": 1331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1336,
                      "end": 1344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1336,
                    "end": 1344
                  }
                }
              ],
              "loc": {
                "start": 1215,
                "end": 1346
              }
            },
            "loc": {
              "start": 1211,
              "end": 1346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1347,
                "end": 1355
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
                      "start": 1362,
                      "end": 1364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1362,
                    "end": 1364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1369,
                      "end": 1379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1369,
                    "end": 1379
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1384,
                      "end": 1394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1384,
                    "end": 1394
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1399,
                      "end": 1409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1399,
                    "end": 1409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1414,
                      "end": 1423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1414,
                    "end": 1423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1428,
                      "end": 1436
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1428,
                    "end": 1436
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1441,
                      "end": 1450
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1441,
                    "end": 1450
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeLanguage",
                    "loc": {
                      "start": 1455,
                      "end": 1467
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1455,
                    "end": 1467
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeType",
                    "loc": {
                      "start": 1472,
                      "end": 1480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1472,
                    "end": 1480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 1485,
                      "end": 1492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1485,
                    "end": 1492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1497,
                      "end": 1509
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1497,
                    "end": 1509
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1514,
                      "end": 1526
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1514,
                    "end": 1526
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "calledByRoutineVersionsCount",
                    "loc": {
                      "start": 1531,
                      "end": 1559
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1531,
                    "end": 1559
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1564,
                      "end": 1577
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1577
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1582,
                      "end": 1604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1582,
                    "end": 1604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1609,
                      "end": 1619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1609,
                    "end": 1619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1624,
                      "end": 1636
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1624,
                    "end": 1636
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1641,
                      "end": 1644
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
                            "start": 1655,
                            "end": 1665
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1655,
                          "end": 1665
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1674,
                            "end": 1681
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1674,
                          "end": 1681
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1690,
                            "end": 1699
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1690,
                          "end": 1699
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1708,
                            "end": 1717
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1708,
                          "end": 1717
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1726,
                            "end": 1735
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1726,
                          "end": 1735
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1744,
                            "end": 1750
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1744,
                          "end": 1750
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1759,
                            "end": 1766
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1759,
                          "end": 1766
                        }
                      }
                    ],
                    "loc": {
                      "start": 1645,
                      "end": 1772
                    }
                  },
                  "loc": {
                    "start": 1641,
                    "end": 1772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1777,
                      "end": 1789
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
                            "start": 1800,
                            "end": 1802
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1800,
                          "end": 1802
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1811,
                            "end": 1819
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1811,
                          "end": 1819
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1828,
                            "end": 1839
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1828,
                          "end": 1839
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 1848,
                            "end": 1860
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1848,
                          "end": 1860
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1869,
                            "end": 1873
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1869,
                          "end": 1873
                        }
                      }
                    ],
                    "loc": {
                      "start": 1790,
                      "end": 1879
                    }
                  },
                  "loc": {
                    "start": 1777,
                    "end": 1879
                  }
                }
              ],
              "loc": {
                "start": 1356,
                "end": 1881
              }
            },
            "loc": {
              "start": 1347,
              "end": 1881
            }
          }
        ],
        "loc": {
          "start": 946,
          "end": 1883
        }
      },
      "loc": {
        "start": 919,
        "end": 1883
      }
    },
    "Code_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_nav",
        "loc": {
          "start": 1893,
          "end": 1901
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 1905,
            "end": 1909
          }
        },
        "loc": {
          "start": 1905,
          "end": 1909
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
                "start": 1912,
                "end": 1914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1912,
              "end": 1914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1915,
                "end": 1924
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1915,
              "end": 1924
            }
          }
        ],
        "loc": {
          "start": 1910,
          "end": 1926
        }
      },
      "loc": {
        "start": 1884,
        "end": 1926
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1936,
          "end": 1946
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1950,
            "end": 1955
          }
        },
        "loc": {
          "start": 1950,
          "end": 1955
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
                "start": 1958,
                "end": 1960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1958,
              "end": 1960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1961,
                "end": 1971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1961,
              "end": 1971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1972,
                "end": 1982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1972,
              "end": 1982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1983,
                "end": 1988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1983,
              "end": 1988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1989,
                "end": 1994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1989,
              "end": 1994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1995,
                "end": 2000
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
                        "start": 2014,
                        "end": 2018
                      }
                    },
                    "loc": {
                      "start": 2014,
                      "end": 2018
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
                            "start": 2032,
                            "end": 2040
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2029,
                          "end": 2040
                        }
                      }
                    ],
                    "loc": {
                      "start": 2019,
                      "end": 2046
                    }
                  },
                  "loc": {
                    "start": 2007,
                    "end": 2046
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
                        "start": 2058,
                        "end": 2062
                      }
                    },
                    "loc": {
                      "start": 2058,
                      "end": 2062
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
                            "start": 2076,
                            "end": 2084
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2073,
                          "end": 2084
                        }
                      }
                    ],
                    "loc": {
                      "start": 2063,
                      "end": 2090
                    }
                  },
                  "loc": {
                    "start": 2051,
                    "end": 2090
                  }
                }
              ],
              "loc": {
                "start": 2001,
                "end": 2092
              }
            },
            "loc": {
              "start": 1995,
              "end": 2092
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2093,
                "end": 2096
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
                      "start": 2103,
                      "end": 2112
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2103,
                    "end": 2112
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2117,
                      "end": 2126
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2117,
                    "end": 2126
                  }
                }
              ],
              "loc": {
                "start": 2097,
                "end": 2128
              }
            },
            "loc": {
              "start": 2093,
              "end": 2128
            }
          }
        ],
        "loc": {
          "start": 1956,
          "end": 2130
        }
      },
      "loc": {
        "start": 1927,
        "end": 2130
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 2140,
          "end": 2149
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2153,
            "end": 2157
          }
        },
        "loc": {
          "start": 2153,
          "end": 2157
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
                "start": 2160,
                "end": 2162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2160,
              "end": 2162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2163,
                "end": 2173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2163,
              "end": 2173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2174,
                "end": 2184
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2174,
              "end": 2184
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2185,
                "end": 2194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2185,
              "end": 2194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2195,
                "end": 2206
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2195,
              "end": 2206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2207,
                "end": 2213
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
                      "start": 2223,
                      "end": 2233
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2220,
                    "end": 2233
                  }
                }
              ],
              "loc": {
                "start": 2214,
                "end": 2235
              }
            },
            "loc": {
              "start": 2207,
              "end": 2235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2236,
                "end": 2241
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
                        "start": 2255,
                        "end": 2259
                      }
                    },
                    "loc": {
                      "start": 2255,
                      "end": 2259
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
                            "start": 2273,
                            "end": 2281
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2270,
                          "end": 2281
                        }
                      }
                    ],
                    "loc": {
                      "start": 2260,
                      "end": 2287
                    }
                  },
                  "loc": {
                    "start": 2248,
                    "end": 2287
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
                        "start": 2299,
                        "end": 2303
                      }
                    },
                    "loc": {
                      "start": 2299,
                      "end": 2303
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
                            "start": 2317,
                            "end": 2325
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2314,
                          "end": 2325
                        }
                      }
                    ],
                    "loc": {
                      "start": 2304,
                      "end": 2331
                    }
                  },
                  "loc": {
                    "start": 2292,
                    "end": 2331
                  }
                }
              ],
              "loc": {
                "start": 2242,
                "end": 2333
              }
            },
            "loc": {
              "start": 2236,
              "end": 2333
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2334,
                "end": 2345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2334,
              "end": 2345
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2346,
                "end": 2360
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2346,
              "end": 2360
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2361,
                "end": 2366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2361,
              "end": 2366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2367,
                "end": 2376
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2367,
              "end": 2376
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2377,
                "end": 2381
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
                      "start": 2391,
                      "end": 2399
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2388,
                    "end": 2399
                  }
                }
              ],
              "loc": {
                "start": 2382,
                "end": 2401
              }
            },
            "loc": {
              "start": 2377,
              "end": 2401
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2402,
                "end": 2416
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2402,
              "end": 2416
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2417,
                "end": 2422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2417,
              "end": 2422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2423,
                "end": 2426
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
                      "start": 2433,
                      "end": 2442
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2433,
                    "end": 2442
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2447,
                      "end": 2458
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2447,
                    "end": 2458
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2463,
                      "end": 2474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2463,
                    "end": 2474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2479,
                      "end": 2488
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2479,
                    "end": 2488
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2493,
                      "end": 2500
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2493,
                    "end": 2500
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2505,
                      "end": 2513
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2505,
                    "end": 2513
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2518,
                      "end": 2530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2518,
                    "end": 2530
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2535,
                      "end": 2543
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2535,
                    "end": 2543
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2548,
                      "end": 2556
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2548,
                    "end": 2556
                  }
                }
              ],
              "loc": {
                "start": 2427,
                "end": 2558
              }
            },
            "loc": {
              "start": 2423,
              "end": 2558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 2559,
                "end": 2567
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
                      "start": 2574,
                      "end": 2576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2574,
                    "end": 2576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2581,
                      "end": 2591
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2581,
                    "end": 2591
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2596,
                      "end": 2606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2596,
                    "end": 2606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 2611,
                      "end": 2619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2611,
                    "end": 2619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 2624,
                      "end": 2633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2624,
                    "end": 2633
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2638,
                      "end": 2650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2638,
                    "end": 2650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 2655,
                      "end": 2667
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2655,
                    "end": 2667
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 2672,
                      "end": 2684
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2672,
                    "end": 2684
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2689,
                      "end": 2692
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
                            "start": 2703,
                            "end": 2713
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2703,
                          "end": 2713
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 2722,
                            "end": 2729
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2722,
                          "end": 2729
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2738,
                            "end": 2747
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2738,
                          "end": 2747
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2756,
                            "end": 2765
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2756,
                          "end": 2765
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2774,
                            "end": 2783
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2774,
                          "end": 2783
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 2792,
                            "end": 2798
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2792,
                          "end": 2798
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2807,
                            "end": 2814
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2807,
                          "end": 2814
                        }
                      }
                    ],
                    "loc": {
                      "start": 2693,
                      "end": 2820
                    }
                  },
                  "loc": {
                    "start": 2689,
                    "end": 2820
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2825,
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
                          "value": "id",
                          "loc": {
                            "start": 2848,
                            "end": 2850
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2848,
                          "end": 2850
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2859,
                            "end": 2867
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2859,
                          "end": 2867
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2876,
                            "end": 2887
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2876,
                          "end": 2887
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2896,
                            "end": 2900
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2896,
                          "end": 2900
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 2909,
                            "end": 2914
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
                                  "start": 2929,
                                  "end": 2931
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2929,
                                "end": 2931
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 2944,
                                  "end": 2953
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2944,
                                "end": 2953
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 2966,
                                  "end": 2970
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2966,
                                "end": 2970
                              }
                            }
                          ],
                          "loc": {
                            "start": 2915,
                            "end": 2980
                          }
                        },
                        "loc": {
                          "start": 2909,
                          "end": 2980
                        }
                      }
                    ],
                    "loc": {
                      "start": 2838,
                      "end": 2986
                    }
                  },
                  "loc": {
                    "start": 2825,
                    "end": 2986
                  }
                }
              ],
              "loc": {
                "start": 2568,
                "end": 2988
              }
            },
            "loc": {
              "start": 2559,
              "end": 2988
            }
          }
        ],
        "loc": {
          "start": 2158,
          "end": 2990
        }
      },
      "loc": {
        "start": 2131,
        "end": 2990
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 3000,
          "end": 3008
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 3012,
            "end": 3016
          }
        },
        "loc": {
          "start": 3012,
          "end": 3016
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
                "start": 3019,
                "end": 3021
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3019,
              "end": 3021
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3022,
                "end": 3031
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3022,
              "end": 3031
            }
          }
        ],
        "loc": {
          "start": 3017,
          "end": 3033
        }
      },
      "loc": {
        "start": 2991,
        "end": 3033
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 3043,
          "end": 3055
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3059,
            "end": 3066
          }
        },
        "loc": {
          "start": 3059,
          "end": 3066
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
                "start": 3069,
                "end": 3071
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3069,
              "end": 3071
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3072,
                "end": 3082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3072,
              "end": 3082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3083,
                "end": 3093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3083,
              "end": 3093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3094,
                "end": 3103
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3094,
              "end": 3103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3104,
                "end": 3115
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3104,
              "end": 3115
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3116,
                "end": 3122
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
                      "start": 3132,
                      "end": 3142
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3129,
                    "end": 3142
                  }
                }
              ],
              "loc": {
                "start": 3123,
                "end": 3144
              }
            },
            "loc": {
              "start": 3116,
              "end": 3144
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3145,
                "end": 3150
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
                        "start": 3164,
                        "end": 3168
                      }
                    },
                    "loc": {
                      "start": 3164,
                      "end": 3168
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
                            "start": 3182,
                            "end": 3190
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3179,
                          "end": 3190
                        }
                      }
                    ],
                    "loc": {
                      "start": 3169,
                      "end": 3196
                    }
                  },
                  "loc": {
                    "start": 3157,
                    "end": 3196
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
                        "start": 3208,
                        "end": 3212
                      }
                    },
                    "loc": {
                      "start": 3208,
                      "end": 3212
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
                            "start": 3226,
                            "end": 3234
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3223,
                          "end": 3234
                        }
                      }
                    ],
                    "loc": {
                      "start": 3213,
                      "end": 3240
                    }
                  },
                  "loc": {
                    "start": 3201,
                    "end": 3240
                  }
                }
              ],
              "loc": {
                "start": 3151,
                "end": 3242
              }
            },
            "loc": {
              "start": 3145,
              "end": 3242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3243,
                "end": 3254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3243,
              "end": 3254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3255,
                "end": 3269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3255,
              "end": 3269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3270,
                "end": 3275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3270,
              "end": 3275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3276,
                "end": 3285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3276,
              "end": 3285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3286,
                "end": 3290
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
                      "start": 3300,
                      "end": 3308
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3297,
                    "end": 3308
                  }
                }
              ],
              "loc": {
                "start": 3291,
                "end": 3310
              }
            },
            "loc": {
              "start": 3286,
              "end": 3310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3311,
                "end": 3325
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3311,
              "end": 3325
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3326,
                "end": 3331
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3326,
              "end": 3331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3332,
                "end": 3335
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
                      "start": 3342,
                      "end": 3351
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3342,
                    "end": 3351
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
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
                    "value": "canTransfer",
                    "loc": {
                      "start": 3372,
                      "end": 3383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3372,
                    "end": 3383
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3388,
                      "end": 3397
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3388,
                    "end": 3397
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3402,
                      "end": 3409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3402,
                    "end": 3409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3414,
                      "end": 3422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3414,
                    "end": 3422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3427,
                      "end": 3439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3427,
                    "end": 3439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3444,
                      "end": 3452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3444,
                    "end": 3452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3457,
                      "end": 3465
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3457,
                    "end": 3465
                  }
                }
              ],
              "loc": {
                "start": 3336,
                "end": 3467
              }
            },
            "loc": {
              "start": 3332,
              "end": 3467
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 3468,
                "end": 3476
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
                      "start": 3483,
                      "end": 3485
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3483,
                    "end": 3485
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3490,
                      "end": 3500
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3490,
                    "end": 3500
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3505,
                      "end": 3515
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3505,
                    "end": 3515
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3520,
                      "end": 3536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3520,
                    "end": 3536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3541,
                      "end": 3549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3541,
                    "end": 3549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3554,
                      "end": 3563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3554,
                    "end": 3563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3568,
                      "end": 3580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3568,
                    "end": 3580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3585,
                      "end": 3601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3585,
                    "end": 3601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3606,
                      "end": 3616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3606,
                    "end": 3616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3621,
                      "end": 3633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3621,
                    "end": 3633
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3638,
                      "end": 3650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3638,
                    "end": 3650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3655,
                      "end": 3667
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
                            "start": 3678,
                            "end": 3680
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3678,
                          "end": 3680
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3689,
                            "end": 3697
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3689,
                          "end": 3697
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3706,
                            "end": 3717
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3706,
                          "end": 3717
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3726,
                            "end": 3730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3726,
                          "end": 3730
                        }
                      }
                    ],
                    "loc": {
                      "start": 3668,
                      "end": 3736
                    }
                  },
                  "loc": {
                    "start": 3655,
                    "end": 3736
                  }
                }
              ],
              "loc": {
                "start": 3477,
                "end": 3738
              }
            },
            "loc": {
              "start": 3468,
              "end": 3738
            }
          }
        ],
        "loc": {
          "start": 3067,
          "end": 3740
        }
      },
      "loc": {
        "start": 3034,
        "end": 3740
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3750,
          "end": 3761
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3765,
            "end": 3772
          }
        },
        "loc": {
          "start": 3765,
          "end": 3772
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
                "start": 3775,
                "end": 3777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3775,
              "end": 3777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3778,
                "end": 3787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3778,
              "end": 3787
            }
          }
        ],
        "loc": {
          "start": 3773,
          "end": 3789
        }
      },
      "loc": {
        "start": 3741,
        "end": 3789
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3799,
          "end": 3812
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3816,
            "end": 3824
          }
        },
        "loc": {
          "start": 3816,
          "end": 3824
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
              "value": "created_at",
              "loc": {
                "start": 3830,
                "end": 3840
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3830,
              "end": 3840
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3841,
                "end": 3851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3841,
              "end": 3851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3852,
                "end": 3861
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
                      "start": 3868,
                      "end": 3870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3868,
                    "end": 3870
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3875,
                      "end": 3885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3875,
                    "end": 3885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3890,
                      "end": 3900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3890,
                    "end": 3900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3905,
                      "end": 3916
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3905,
                    "end": 3916
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3921,
                      "end": 3927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3921,
                    "end": 3927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3932,
                      "end": 3937
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3932,
                    "end": 3937
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBotDepictingPerson",
                    "loc": {
                      "start": 3942,
                      "end": 3962
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3942,
                    "end": 3962
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3967,
                      "end": 3971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3967,
                    "end": 3971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3976,
                      "end": 3988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3976,
                    "end": 3988
                  }
                }
              ],
              "loc": {
                "start": 3862,
                "end": 3990
              }
            },
            "loc": {
              "start": 3852,
              "end": 3990
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 3991,
                "end": 4008
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3991,
              "end": 4008
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4009,
                "end": 4018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4009,
              "end": 4018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4019,
                "end": 4024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4019,
              "end": 4024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4025,
                "end": 4034
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4025,
              "end": 4034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4035,
                "end": 4047
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4035,
              "end": 4047
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4048,
                "end": 4061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4048,
              "end": 4061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4062,
                "end": 4074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4062,
              "end": 4074
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4075,
                "end": 4084
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
                        "start": 4098,
                        "end": 4101
                      }
                    },
                    "loc": {
                      "start": 4098,
                      "end": 4101
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
                            "start": 4115,
                            "end": 4122
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4112,
                          "end": 4122
                        }
                      }
                    ],
                    "loc": {
                      "start": 4102,
                      "end": 4128
                    }
                  },
                  "loc": {
                    "start": 4091,
                    "end": 4128
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
                        "start": 4140,
                        "end": 4144
                      }
                    },
                    "loc": {
                      "start": 4140,
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
                          "value": "Code_nav",
                          "loc": {
                            "start": 4158,
                            "end": 4166
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4155,
                          "end": 4166
                        }
                      }
                    ],
                    "loc": {
                      "start": 4145,
                      "end": 4172
                    }
                  },
                  "loc": {
                    "start": 4133,
                    "end": 4172
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
                        "start": 4184,
                        "end": 4188
                      }
                    },
                    "loc": {
                      "start": 4184,
                      "end": 4188
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
                            "start": 4202,
                            "end": 4210
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4199,
                          "end": 4210
                        }
                      }
                    ],
                    "loc": {
                      "start": 4189,
                      "end": 4216
                    }
                  },
                  "loc": {
                    "start": 4177,
                    "end": 4216
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
                        "start": 4228,
                        "end": 4235
                      }
                    },
                    "loc": {
                      "start": 4228,
                      "end": 4235
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
                            "start": 4249,
                            "end": 4260
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4246,
                          "end": 4260
                        }
                      }
                    ],
                    "loc": {
                      "start": 4236,
                      "end": 4266
                    }
                  },
                  "loc": {
                    "start": 4221,
                    "end": 4266
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
                        "start": 4278,
                        "end": 4285
                      }
                    },
                    "loc": {
                      "start": 4278,
                      "end": 4285
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
                            "start": 4299,
                            "end": 4310
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4296,
                          "end": 4310
                        }
                      }
                    ],
                    "loc": {
                      "start": 4286,
                      "end": 4316
                    }
                  },
                  "loc": {
                    "start": 4271,
                    "end": 4316
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
                        "start": 4328,
                        "end": 4336
                      }
                    },
                    "loc": {
                      "start": 4328,
                      "end": 4336
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
                            "start": 4350,
                            "end": 4362
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4347,
                          "end": 4362
                        }
                      }
                    ],
                    "loc": {
                      "start": 4337,
                      "end": 4368
                    }
                  },
                  "loc": {
                    "start": 4321,
                    "end": 4368
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
                        "start": 4380,
                        "end": 4384
                      }
                    },
                    "loc": {
                      "start": 4380,
                      "end": 4384
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
                            "start": 4398,
                            "end": 4406
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4395,
                          "end": 4406
                        }
                      }
                    ],
                    "loc": {
                      "start": 4385,
                      "end": 4412
                    }
                  },
                  "loc": {
                    "start": 4373,
                    "end": 4412
                  }
                }
              ],
              "loc": {
                "start": 4085,
                "end": 4414
              }
            },
            "loc": {
              "start": 4075,
              "end": 4414
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4415,
                "end": 4419
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
                      "start": 4429,
                      "end": 4437
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4426,
                    "end": 4437
                  }
                }
              ],
              "loc": {
                "start": 4420,
                "end": 4439
              }
            },
            "loc": {
              "start": 4415,
              "end": 4439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4440,
                "end": 4443
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
                      "start": 4450,
                      "end": 4458
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4450,
                    "end": 4458
                  }
                }
              ],
              "loc": {
                "start": 4444,
                "end": 4460
              }
            },
            "loc": {
              "start": 4440,
              "end": 4460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 4461,
                "end": 4473
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
                      "start": 4480,
                      "end": 4482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4480,
                    "end": 4482
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4487,
                      "end": 4495
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4487,
                    "end": 4495
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4500,
                      "end": 4511
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4500,
                    "end": 4511
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4516,
                      "end": 4520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4516,
                    "end": 4520
                  }
                }
              ],
              "loc": {
                "start": 4474,
                "end": 4522
              }
            },
            "loc": {
              "start": 4461,
              "end": 4522
            }
          }
        ],
        "loc": {
          "start": 3825,
          "end": 4524
        }
      },
      "loc": {
        "start": 3790,
        "end": 4524
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4534,
          "end": 4546
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4550,
            "end": 4557
          }
        },
        "loc": {
          "start": 4550,
          "end": 4557
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
                "start": 4560,
                "end": 4562
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4560,
              "end": 4562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4563,
                "end": 4573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4563,
              "end": 4573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4574,
                "end": 4584
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4574,
              "end": 4584
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 4585,
                "end": 4595
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4585,
              "end": 4595
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4596,
                "end": 4605
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4596,
              "end": 4605
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 4606,
                "end": 4617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4606,
              "end": 4617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 4618,
                "end": 4624
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
                      "start": 4634,
                      "end": 4644
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4631,
                    "end": 4644
                  }
                }
              ],
              "loc": {
                "start": 4625,
                "end": 4646
              }
            },
            "loc": {
              "start": 4618,
              "end": 4646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 4647,
                "end": 4652
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
                        "start": 4666,
                        "end": 4670
                      }
                    },
                    "loc": {
                      "start": 4666,
                      "end": 4670
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
                            "start": 4684,
                            "end": 4692
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4681,
                          "end": 4692
                        }
                      }
                    ],
                    "loc": {
                      "start": 4671,
                      "end": 4698
                    }
                  },
                  "loc": {
                    "start": 4659,
                    "end": 4698
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
                        "start": 4710,
                        "end": 4714
                      }
                    },
                    "loc": {
                      "start": 4710,
                      "end": 4714
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
                            "start": 4728,
                            "end": 4736
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4725,
                          "end": 4736
                        }
                      }
                    ],
                    "loc": {
                      "start": 4715,
                      "end": 4742
                    }
                  },
                  "loc": {
                    "start": 4703,
                    "end": 4742
                  }
                }
              ],
              "loc": {
                "start": 4653,
                "end": 4744
              }
            },
            "loc": {
              "start": 4647,
              "end": 4744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 4745,
                "end": 4756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4745,
              "end": 4756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 4757,
                "end": 4771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4757,
              "end": 4771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4772,
                "end": 4777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4772,
              "end": 4777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4778,
                "end": 4787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4778,
              "end": 4787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4788,
                "end": 4792
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
                      "start": 4802,
                      "end": 4810
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4799,
                    "end": 4810
                  }
                }
              ],
              "loc": {
                "start": 4793,
                "end": 4812
              }
            },
            "loc": {
              "start": 4788,
              "end": 4812
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 4813,
                "end": 4827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4813,
              "end": 4827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 4828,
                "end": 4833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4828,
              "end": 4833
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4834,
                "end": 4837
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
                      "start": 4844,
                      "end": 4854
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4844,
                    "end": 4854
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 4859,
                      "end": 4868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4859,
                    "end": 4868
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 4873,
                      "end": 4884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4873,
                    "end": 4884
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 4889,
                      "end": 4898
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4889,
                    "end": 4898
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 4903,
                      "end": 4910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4903,
                    "end": 4910
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 4915,
                      "end": 4923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4915,
                    "end": 4923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 4928,
                      "end": 4940
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4928,
                    "end": 4940
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 4945,
                      "end": 4953
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4945,
                    "end": 4953
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 4958,
                      "end": 4966
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4958,
                    "end": 4966
                  }
                }
              ],
              "loc": {
                "start": 4838,
                "end": 4968
              }
            },
            "loc": {
              "start": 4834,
              "end": 4968
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4969,
                "end": 4977
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
                      "start": 4984,
                      "end": 4986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4984,
                    "end": 4986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4991,
                      "end": 5001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4991,
                    "end": 5001
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5006,
                      "end": 5016
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5006,
                    "end": 5016
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 5021,
                      "end": 5032
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5021,
                    "end": 5032
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 5037,
                      "end": 5050
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5037,
                    "end": 5050
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5055,
                      "end": 5065
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5055,
                    "end": 5065
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5070,
                      "end": 5079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5070,
                    "end": 5079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5084,
                      "end": 5092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5084,
                    "end": 5092
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5097,
                      "end": 5106
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5097,
                    "end": 5106
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 5111,
                      "end": 5122
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5111,
                    "end": 5122
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 5127,
                      "end": 5137
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5127,
                    "end": 5137
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 5142,
                      "end": 5154
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5142,
                    "end": 5154
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 5159,
                      "end": 5173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5159,
                    "end": 5173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5178,
                      "end": 5190
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5178,
                    "end": 5190
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5195,
                      "end": 5207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5195,
                    "end": 5207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5212,
                      "end": 5225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5212,
                    "end": 5225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5230,
                      "end": 5252
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5230,
                    "end": 5252
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5257,
                      "end": 5267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5257,
                    "end": 5267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 5272,
                      "end": 5283
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5272,
                    "end": 5283
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 5288,
                      "end": 5298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5288,
                    "end": 5298
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 5303,
                      "end": 5317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5303,
                    "end": 5317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 5322,
                      "end": 5334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5322,
                    "end": 5334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5339,
                      "end": 5351
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5339,
                    "end": 5351
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5356,
                      "end": 5368
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
                            "start": 5379,
                            "end": 5381
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5379,
                          "end": 5381
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5390,
                            "end": 5398
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5390,
                          "end": 5398
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5407,
                            "end": 5418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5407,
                          "end": 5418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 5427,
                            "end": 5439
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5427,
                          "end": 5439
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5448,
                            "end": 5452
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5448,
                          "end": 5452
                        }
                      }
                    ],
                    "loc": {
                      "start": 5369,
                      "end": 5458
                    }
                  },
                  "loc": {
                    "start": 5356,
                    "end": 5458
                  }
                }
              ],
              "loc": {
                "start": 4978,
                "end": 5460
              }
            },
            "loc": {
              "start": 4969,
              "end": 5460
            }
          }
        ],
        "loc": {
          "start": 4558,
          "end": 5462
        }
      },
      "loc": {
        "start": 4525,
        "end": 5462
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5472,
          "end": 5483
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5487,
            "end": 5494
          }
        },
        "loc": {
          "start": 5487,
          "end": 5494
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
                "start": 5497,
                "end": 5499
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5497,
              "end": 5499
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5500,
                "end": 5510
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5500,
              "end": 5510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5511,
                "end": 5520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5511,
              "end": 5520
            }
          }
        ],
        "loc": {
          "start": 5495,
          "end": 5522
        }
      },
      "loc": {
        "start": 5463,
        "end": 5522
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 5532,
          "end": 5545
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 5549,
            "end": 5557
          }
        },
        "loc": {
          "start": 5549,
          "end": 5557
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
                "start": 5560,
                "end": 5562
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5560,
              "end": 5562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5563,
                "end": 5573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5563,
              "end": 5573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5574,
                "end": 5584
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5574,
              "end": 5584
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5585,
                "end": 5594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5585,
              "end": 5594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5595,
                "end": 5606
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5595,
              "end": 5606
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5607,
                "end": 5613
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
                      "start": 5623,
                      "end": 5633
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5620,
                    "end": 5633
                  }
                }
              ],
              "loc": {
                "start": 5614,
                "end": 5635
              }
            },
            "loc": {
              "start": 5607,
              "end": 5635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5636,
                "end": 5641
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
                        "start": 5655,
                        "end": 5659
                      }
                    },
                    "loc": {
                      "start": 5655,
                      "end": 5659
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
                            "start": 5673,
                            "end": 5681
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5670,
                          "end": 5681
                        }
                      }
                    ],
                    "loc": {
                      "start": 5660,
                      "end": 5687
                    }
                  },
                  "loc": {
                    "start": 5648,
                    "end": 5687
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
                        "start": 5699,
                        "end": 5703
                      }
                    },
                    "loc": {
                      "start": 5699,
                      "end": 5703
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
                            "start": 5717,
                            "end": 5725
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5714,
                          "end": 5725
                        }
                      }
                    ],
                    "loc": {
                      "start": 5704,
                      "end": 5731
                    }
                  },
                  "loc": {
                    "start": 5692,
                    "end": 5731
                  }
                }
              ],
              "loc": {
                "start": 5642,
                "end": 5733
              }
            },
            "loc": {
              "start": 5636,
              "end": 5733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5734,
                "end": 5745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5734,
              "end": 5745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5746,
                "end": 5760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5746,
              "end": 5760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5761,
                "end": 5766
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5761,
              "end": 5766
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5767,
                "end": 5776
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5767,
              "end": 5776
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5777,
                "end": 5781
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
                      "start": 5791,
                      "end": 5799
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5788,
                    "end": 5799
                  }
                }
              ],
              "loc": {
                "start": 5782,
                "end": 5801
              }
            },
            "loc": {
              "start": 5777,
              "end": 5801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5802,
                "end": 5816
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5802,
              "end": 5816
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5817,
                "end": 5822
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5817,
              "end": 5822
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5823,
                "end": 5826
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
                      "start": 5833,
                      "end": 5842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5833,
                    "end": 5842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5847,
                      "end": 5858
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5847,
                    "end": 5858
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 5863,
                      "end": 5874
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5863,
                    "end": 5874
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5879,
                      "end": 5888
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5879,
                    "end": 5888
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5893,
                      "end": 5900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5893,
                    "end": 5900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5905,
                      "end": 5913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5905,
                    "end": 5913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5918,
                      "end": 5930
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5918,
                    "end": 5930
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5935,
                      "end": 5943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5935,
                    "end": 5943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5948,
                      "end": 5956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5948,
                    "end": 5956
                  }
                }
              ],
              "loc": {
                "start": 5827,
                "end": 5958
              }
            },
            "loc": {
              "start": 5823,
              "end": 5958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5959,
                "end": 5967
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
                      "start": 5974,
                      "end": 5976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5974,
                    "end": 5976
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5981,
                      "end": 5991
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5981,
                    "end": 5991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5996,
                      "end": 6006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5996,
                    "end": 6006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6011,
                      "end": 6021
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6011,
                    "end": 6021
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6026,
                      "end": 6032
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6026,
                    "end": 6032
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6037,
                      "end": 6045
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6037,
                    "end": 6045
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6050,
                      "end": 6059
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6050,
                    "end": 6059
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6064,
                      "end": 6071
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6064,
                    "end": 6071
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6076,
                      "end": 6088
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6076,
                    "end": 6088
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6093,
                      "end": 6098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6093,
                    "end": 6098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6103,
                      "end": 6106
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6103,
                    "end": 6106
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6111,
                      "end": 6123
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6111,
                    "end": 6123
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6128,
                      "end": 6140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6128,
                    "end": 6140
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6145,
                      "end": 6158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6145,
                    "end": 6158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6163,
                      "end": 6185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6163,
                    "end": 6185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6190,
                      "end": 6200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6190,
                    "end": 6200
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6205,
                      "end": 6217
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6205,
                    "end": 6217
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6222,
                      "end": 6225
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
                            "start": 6236,
                            "end": 6246
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6236,
                          "end": 6246
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6255,
                            "end": 6262
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6255,
                          "end": 6262
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6271,
                            "end": 6280
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6271,
                          "end": 6280
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6289,
                            "end": 6298
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6289,
                          "end": 6298
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6307,
                            "end": 6316
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6307,
                          "end": 6316
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6325,
                            "end": 6331
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6325,
                          "end": 6331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6340,
                            "end": 6347
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6340,
                          "end": 6347
                        }
                      }
                    ],
                    "loc": {
                      "start": 6226,
                      "end": 6353
                    }
                  },
                  "loc": {
                    "start": 6222,
                    "end": 6353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6358,
                      "end": 6370
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
                            "start": 6381,
                            "end": 6383
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6381,
                          "end": 6383
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6392,
                            "end": 6400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6392,
                          "end": 6400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6409,
                            "end": 6420
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6409,
                          "end": 6420
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6429,
                            "end": 6441
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6429,
                          "end": 6441
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6450,
                            "end": 6454
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6450,
                          "end": 6454
                        }
                      }
                    ],
                    "loc": {
                      "start": 6371,
                      "end": 6460
                    }
                  },
                  "loc": {
                    "start": 6358,
                    "end": 6460
                  }
                }
              ],
              "loc": {
                "start": 5968,
                "end": 6462
              }
            },
            "loc": {
              "start": 5959,
              "end": 6462
            }
          }
        ],
        "loc": {
          "start": 5558,
          "end": 6464
        }
      },
      "loc": {
        "start": 5523,
        "end": 6464
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 6474,
          "end": 6486
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6490,
            "end": 6498
          }
        },
        "loc": {
          "start": 6490,
          "end": 6498
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
                "start": 6501,
                "end": 6503
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6501,
              "end": 6503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6504,
                "end": 6513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6504,
              "end": 6513
            }
          }
        ],
        "loc": {
          "start": 6499,
          "end": 6515
        }
      },
      "loc": {
        "start": 6465,
        "end": 6515
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6525,
          "end": 6533
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6537,
            "end": 6540
          }
        },
        "loc": {
          "start": 6537,
          "end": 6540
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
                "start": 6543,
                "end": 6545
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6543,
              "end": 6545
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6546,
                "end": 6556
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6546,
              "end": 6556
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6557,
                "end": 6560
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6557,
              "end": 6560
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6561,
                "end": 6570
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6561,
              "end": 6570
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6571,
                "end": 6583
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
                      "start": 6590,
                      "end": 6592
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6590,
                    "end": 6592
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6597,
                      "end": 6605
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6597,
                    "end": 6605
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6610,
                      "end": 6621
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6610,
                    "end": 6621
                  }
                }
              ],
              "loc": {
                "start": 6584,
                "end": 6623
              }
            },
            "loc": {
              "start": 6571,
              "end": 6623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6624,
                "end": 6627
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
                      "start": 6634,
                      "end": 6639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6634,
                    "end": 6639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6644,
                      "end": 6656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6644,
                    "end": 6656
                  }
                }
              ],
              "loc": {
                "start": 6628,
                "end": 6658
              }
            },
            "loc": {
              "start": 6624,
              "end": 6658
            }
          }
        ],
        "loc": {
          "start": 6541,
          "end": 6660
        }
      },
      "loc": {
        "start": 6516,
        "end": 6660
      }
    },
    "Team_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_list",
        "loc": {
          "start": 6670,
          "end": 6679
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 6683,
            "end": 6687
          }
        },
        "loc": {
          "start": 6683,
          "end": 6687
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
                "start": 6690,
                "end": 6692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6690,
              "end": 6692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 6693,
                "end": 6704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6693,
              "end": 6704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 6705,
                "end": 6711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6705,
              "end": 6711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6712,
                "end": 6722
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6712,
              "end": 6722
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6723,
                "end": 6733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6723,
              "end": 6733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 6734,
                "end": 6752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6734,
              "end": 6752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6753,
                "end": 6762
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6753,
              "end": 6762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6763,
                "end": 6776
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6763,
              "end": 6776
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 6777,
                "end": 6789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6777,
              "end": 6789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 6790,
                "end": 6802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6790,
              "end": 6802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6803,
                "end": 6815
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6803,
              "end": 6815
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6816,
                "end": 6825
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6816,
              "end": 6825
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6826,
                "end": 6830
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
                      "start": 6840,
                      "end": 6848
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6837,
                    "end": 6848
                  }
                }
              ],
              "loc": {
                "start": 6831,
                "end": 6850
              }
            },
            "loc": {
              "start": 6826,
              "end": 6850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6851,
                "end": 6863
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
                      "start": 6870,
                      "end": 6872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6870,
                    "end": 6872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6877,
                      "end": 6885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6877,
                    "end": 6885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 6890,
                      "end": 6893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6890,
                    "end": 6893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6898,
                      "end": 6902
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6898,
                    "end": 6902
                  }
                }
              ],
              "loc": {
                "start": 6864,
                "end": 6904
              }
            },
            "loc": {
              "start": 6851,
              "end": 6904
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6905,
                "end": 6908
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
                      "start": 6915,
                      "end": 6928
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6915,
                    "end": 6928
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6933,
                      "end": 6942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6933,
                    "end": 6942
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6947,
                      "end": 6958
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6947,
                    "end": 6958
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6963,
                      "end": 6972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6963,
                    "end": 6972
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6977,
                      "end": 6986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6977,
                    "end": 6986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6991,
                      "end": 6998
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6991,
                    "end": 6998
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7003,
                      "end": 7015
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7003,
                    "end": 7015
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7020,
                      "end": 7028
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7020,
                    "end": 7028
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7033,
                      "end": 7047
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
                            "start": 7058,
                            "end": 7060
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7058,
                          "end": 7060
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
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
                          "value": "updated_at",
                          "loc": {
                            "start": 7088,
                            "end": 7098
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7088,
                          "end": 7098
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7107,
                            "end": 7114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7107,
                          "end": 7114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7123,
                            "end": 7134
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7123,
                          "end": 7134
                        }
                      }
                    ],
                    "loc": {
                      "start": 7048,
                      "end": 7140
                    }
                  },
                  "loc": {
                    "start": 7033,
                    "end": 7140
                  }
                }
              ],
              "loc": {
                "start": 6909,
                "end": 7142
              }
            },
            "loc": {
              "start": 6905,
              "end": 7142
            }
          }
        ],
        "loc": {
          "start": 6688,
          "end": 7144
        }
      },
      "loc": {
        "start": 6661,
        "end": 7144
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 7154,
          "end": 7162
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 7166,
            "end": 7170
          }
        },
        "loc": {
          "start": 7166,
          "end": 7170
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
                "start": 7173,
                "end": 7175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7173,
              "end": 7175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7176,
                "end": 7187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7176,
              "end": 7187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7188,
                "end": 7194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7188,
              "end": 7194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7195,
                "end": 7207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7195,
              "end": 7207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7208,
                "end": 7211
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
                      "start": 7218,
                      "end": 7231
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7218,
                    "end": 7231
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7236,
                      "end": 7245
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7236,
                    "end": 7245
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7250,
                      "end": 7261
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7250,
                    "end": 7261
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7266,
                      "end": 7275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7266,
                    "end": 7275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7280,
                      "end": 7289
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7280,
                    "end": 7289
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7294,
                      "end": 7301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7294,
                    "end": 7301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7306,
                      "end": 7318
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7306,
                    "end": 7318
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7323,
                      "end": 7331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7323,
                    "end": 7331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7336,
                      "end": 7350
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
                            "start": 7361,
                            "end": 7363
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7361,
                          "end": 7363
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
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
                          "value": "updated_at",
                          "loc": {
                            "start": 7391,
                            "end": 7401
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7391,
                          "end": 7401
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7410,
                            "end": 7417
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7410,
                          "end": 7417
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7426,
                            "end": 7437
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7426,
                          "end": 7437
                        }
                      }
                    ],
                    "loc": {
                      "start": 7351,
                      "end": 7443
                    }
                  },
                  "loc": {
                    "start": 7336,
                    "end": 7443
                  }
                }
              ],
              "loc": {
                "start": 7212,
                "end": 7445
              }
            },
            "loc": {
              "start": 7208,
              "end": 7445
            }
          }
        ],
        "loc": {
          "start": 7171,
          "end": 7447
        }
      },
      "loc": {
        "start": 7145,
        "end": 7447
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7457,
          "end": 7466
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7470,
            "end": 7474
          }
        },
        "loc": {
          "start": 7470,
          "end": 7474
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
              "value": "created_at",
              "loc": {
                "start": 7480,
                "end": 7490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7480,
              "end": 7490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7491,
                "end": 7501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7491,
              "end": 7501
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7502,
                "end": 7513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7502,
              "end": 7513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7514,
                "end": 7520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7514,
              "end": 7520
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7521,
                "end": 7526
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7521,
              "end": 7526
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7527,
                "end": 7547
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7527,
              "end": 7547
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7548,
                "end": 7552
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7548,
              "end": 7552
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7553,
                "end": 7565
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7553,
              "end": 7565
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7566,
                "end": 7575
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7566,
              "end": 7575
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7576,
                "end": 7596
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7576,
              "end": 7596
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7597,
                "end": 7600
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
                      "start": 7607,
                      "end": 7616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7607,
                    "end": 7616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7621,
                      "end": 7630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7621,
                    "end": 7630
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7635,
                      "end": 7644
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7635,
                    "end": 7644
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7649,
                      "end": 7661
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7649,
                    "end": 7661
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7666,
                      "end": 7674
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7666,
                    "end": 7674
                  }
                }
              ],
              "loc": {
                "start": 7601,
                "end": 7676
              }
            },
            "loc": {
              "start": 7597,
              "end": 7676
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7677,
                "end": 7689
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
                      "start": 7696,
                      "end": 7698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7696,
                    "end": 7698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7703,
                      "end": 7711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7703,
                    "end": 7711
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7716,
                      "end": 7719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7716,
                    "end": 7719
                  }
                }
              ],
              "loc": {
                "start": 7690,
                "end": 7721
              }
            },
            "loc": {
              "start": 7677,
              "end": 7721
            }
          }
        ],
        "loc": {
          "start": 7475,
          "end": 7723
        }
      },
      "loc": {
        "start": 7448,
        "end": 7723
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7733,
          "end": 7741
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7745,
            "end": 7749
          }
        },
        "loc": {
          "start": 7745,
          "end": 7749
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
                "start": 7752,
                "end": 7754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7752,
              "end": 7754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7755,
                "end": 7765
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7755,
              "end": 7765
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7766,
                "end": 7776
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7766,
              "end": 7776
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7777,
                "end": 7788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7777,
              "end": 7788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7789,
                "end": 7795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7789,
              "end": 7795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7796,
                "end": 7801
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7796,
              "end": 7801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7802,
                "end": 7822
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7802,
              "end": 7822
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7823,
                "end": 7827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7823,
              "end": 7827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7828,
                "end": 7840
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7828,
              "end": 7840
            }
          }
        ],
        "loc": {
          "start": 7750,
          "end": 7842
        }
      },
      "loc": {
        "start": 7724,
        "end": 7842
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
        "start": 7850,
        "end": 7858
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
              "start": 7860,
              "end": 7865
            }
          },
          "loc": {
            "start": 7859,
            "end": 7865
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
                "start": 7867,
                "end": 7885
              }
            },
            "loc": {
              "start": 7867,
              "end": 7885
            }
          },
          "loc": {
            "start": 7867,
            "end": 7886
          }
        },
        "directives": [],
        "loc": {
          "start": 7859,
          "end": 7886
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
              "start": 7892,
              "end": 7900
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7901,
                  "end": 7906
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7909,
                    "end": 7914
                  }
                },
                "loc": {
                  "start": 7908,
                  "end": 7914
                }
              },
              "loc": {
                "start": 7901,
                "end": 7914
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
                    "start": 7922,
                    "end": 7927
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
                          "start": 7938,
                          "end": 7944
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 7938,
                        "end": 7944
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 7953,
                          "end": 7957
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
                                  "start": 7979,
                                  "end": 7982
                                }
                              },
                              "loc": {
                                "start": 7979,
                                "end": 7982
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
                                      "start": 8004,
                                      "end": 8012
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8001,
                                    "end": 8012
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7983,
                                "end": 8026
                              }
                            },
                            "loc": {
                              "start": 7972,
                              "end": 8026
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
                                  "start": 8046,
                                  "end": 8050
                                }
                              },
                              "loc": {
                                "start": 8046,
                                "end": 8050
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
                                      "start": 8072,
                                      "end": 8081
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8069,
                                    "end": 8081
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8051,
                                "end": 8095
                              }
                            },
                            "loc": {
                              "start": 8039,
                              "end": 8095
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
                                  "start": 8115,
                                  "end": 8119
                                }
                              },
                              "loc": {
                                "start": 8115,
                                "end": 8119
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
                                      "start": 8141,
                                      "end": 8150
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8138,
                                    "end": 8150
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8120,
                                "end": 8164
                              }
                            },
                            "loc": {
                              "start": 8108,
                              "end": 8164
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
                                  "start": 8184,
                                  "end": 8191
                                }
                              },
                              "loc": {
                                "start": 8184,
                                "end": 8191
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
                                      "start": 8213,
                                      "end": 8225
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8210,
                                    "end": 8225
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8192,
                                "end": 8239
                              }
                            },
                            "loc": {
                              "start": 8177,
                              "end": 8239
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
                                  "start": 8259,
                                  "end": 8267
                                }
                              },
                              "loc": {
                                "start": 8259,
                                "end": 8267
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
                                      "start": 8289,
                                      "end": 8302
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8286,
                                    "end": 8302
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8268,
                                "end": 8316
                              }
                            },
                            "loc": {
                              "start": 8252,
                              "end": 8316
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
                                  "start": 8336,
                                  "end": 8343
                                }
                              },
                              "loc": {
                                "start": 8336,
                                "end": 8343
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
                                      "start": 8365,
                                      "end": 8377
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8362,
                                    "end": 8377
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8344,
                                "end": 8391
                              }
                            },
                            "loc": {
                              "start": 8329,
                              "end": 8391
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
                                  "start": 8411,
                                  "end": 8419
                                }
                              },
                              "loc": {
                                "start": 8411,
                                "end": 8419
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
                                      "start": 8441,
                                      "end": 8454
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8438,
                                    "end": 8454
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8420,
                                "end": 8468
                              }
                            },
                            "loc": {
                              "start": 8404,
                              "end": 8468
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
                                  "start": 8488,
                                  "end": 8492
                                }
                              },
                              "loc": {
                                "start": 8488,
                                "end": 8492
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
                                      "start": 8514,
                                      "end": 8523
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8511,
                                    "end": 8523
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8493,
                                "end": 8537
                              }
                            },
                            "loc": {
                              "start": 8481,
                              "end": 8537
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
                                  "start": 8557,
                                  "end": 8561
                                }
                              },
                              "loc": {
                                "start": 8557,
                                "end": 8561
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
                                      "start": 8583,
                                      "end": 8592
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8580,
                                    "end": 8592
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8562,
                                "end": 8606
                              }
                            },
                            "loc": {
                              "start": 8550,
                              "end": 8606
                            }
                          }
                        ],
                        "loc": {
                          "start": 7958,
                          "end": 8616
                        }
                      },
                      "loc": {
                        "start": 7953,
                        "end": 8616
                      }
                    }
                  ],
                  "loc": {
                    "start": 7928,
                    "end": 8622
                  }
                },
                "loc": {
                  "start": 7922,
                  "end": 8622
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8627,
                    "end": 8635
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
                          "start": 8646,
                          "end": 8657
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8646,
                        "end": 8657
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8666,
                          "end": 8678
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8666,
                        "end": 8678
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorCode",
                        "loc": {
                          "start": 8687,
                          "end": 8700
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8687,
                        "end": 8700
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8709,
                          "end": 8722
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8709,
                        "end": 8722
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8731,
                          "end": 8747
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8731,
                        "end": 8747
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8756,
                          "end": 8773
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8756,
                        "end": 8773
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8782,
                          "end": 8798
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8782,
                        "end": 8798
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 8807,
                          "end": 8824
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8807,
                        "end": 8824
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorTeam",
                        "loc": {
                          "start": 8833,
                          "end": 8846
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8833,
                        "end": 8846
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 8855,
                          "end": 8868
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8855,
                        "end": 8868
                      }
                    }
                  ],
                  "loc": {
                    "start": 8636,
                    "end": 8874
                  }
                },
                "loc": {
                  "start": 8627,
                  "end": 8874
                }
              }
            ],
            "loc": {
              "start": 7916,
              "end": 8878
            }
          },
          "loc": {
            "start": 7892,
            "end": 8878
          }
        }
      ],
      "loc": {
        "start": 7888,
        "end": 8880
      }
    },
    "loc": {
      "start": 7844,
      "end": 8880
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
